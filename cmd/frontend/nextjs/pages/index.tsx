import { useState, useRef, useEffect } from 'react';
import Head from 'next/head';
import { SessionRequest, SessionResponse, SSEEvent, StepStatus, FinalResult } from '../types';
import Timeline from '../components/Timeline';
import LessonCard from '../components/LessonCard';
import Sidebar from '../components/Sidebar';
import { ExplanationType } from '../types';
import VisualizationView from '../components/VisualizationView';
import SimpleExplanationView from '../components/SimpleExplanationView';
import ErrorBoundary from '../components/ErrorBoundary';
import ErrorMessage from '../components/ErrorMessage';
import LoadingSkeleton, { TimelineSkeleton } from '../components/LoadingSkeleton';
import RetryButton from '../components/RetryButton';
import BrainPrintCard from '../components/BrainPrintCard';
import { useRetry } from '../hooks/useRetry';
import { getOrchestratorURL } from '../utils/getOrchestratorURL';
import { validateTopic } from '../utils/validation';
import { handleError, isRetryableError } from '../utils/errorHandler';

const STEPS = [
  { id: 'summarizer', name: 'Summarizer', description: 'Analyzing topic and context' },
  { id: 'explainer', name: 'Explainer', description: 'Creating structured lesson' },
  { id: 'visualizer', name: 'Visualizer', description: 'Generating diagrams' },
  { id: 'critic', name: 'Critic', description: 'Reviewing and improving' },
];

export default function Home() {
  const [topic, setTopic] = useState('');
  const [explanationType, setExplanationType] = useState<ExplanationType>('standard');
  const [isLoading, setIsLoading] = useState(false);
  const [isInitialLoading, setIsInitialLoading] = useState(false);
  const [sessionId, setSessionId] = useState<string | null>(null);
  const [steps, setSteps] = useState<StepStatus[]>([]);
  const [finalResult, setFinalResult] = useState<FinalResult | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [validationError, setValidationError] = useState<string | null>(null);
  const [isSSEConnected, setIsSSEConnected] = useState(false);
  const eventSourceRef = useRef<EventSource | null>(null);
  const brainPrintRefreshRef = useRef<(() => void) | null>(null);

  // Retry hook for session creation
  const { execute: retryCreateSession, isRetrying: isRetryingSession } = useRetry(
    async (request: SessionRequest) => {
      const response = await fetch('/api/sessions', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(request),
      });

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({ error: response.statusText }));
        throw new Error(errorData.error || `Failed to create session: ${response.statusText}`);
      }

      return response.json();
    },
    {
      maxRetries: 3,
      retryDelay: 2000,
    }
  );

  // Initialize steps
  useEffect(() => {
    const initialSteps: StepStatus[] = STEPS.map(step => ({
      step: step.id,
      status: 'pending',
      timestamp: new Date().toISOString(),
    }));
    setSteps(initialSteps);
  }, []);

  // Cleanup SSE connection
  useEffect(() => {
    return () => {
      if (eventSourceRef.current) {
        eventSourceRef.current.close();
      }
    };
  }, []);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    
    // Validate topic
    const validation = validateTopic(topic);
    if (!validation.valid) {
      setValidationError(validation.error || 'Invalid topic');
      return;
    }
    setValidationError(null);

    setIsLoading(true);
    setIsInitialLoading(true);
    setError(null);
    setValidationError(null);
    setFinalResult(null);
    setSessionId(null);
    setIsSSEConnected(false);

    try {
      // Create session with explanation type
      const sessionRequest: SessionRequest = { 
        topic: topic.trim(),
        explanation_type: explanationType 
      };
      
      const sessionData = await retryCreateSession(sessionRequest);
      
      if (!sessionData) {
        throw new Error('Failed to create session after retries');
      }

      setSessionId(sessionData.session_id);

      // Reset steps to pending
      const resetSteps: StepStatus[] = STEPS.map(step => ({
        step: step.id,
        status: 'pending',
        timestamp: new Date().toISOString(),
      }));
      setSteps(resetSteps);

      // Connect to SSE
      connectToSSE(sessionData.session_id);
    } catch (err) {
      const errorInfo = handleError(err);
      setError(errorInfo.message);
      setIsLoading(false);
      setIsInitialLoading(false);
    }
  };

  const connectToSSE = (sessionId: string, retryCount = 0) => {
    if (eventSourceRef.current) {
      eventSourceRef.current.close();
    }

    const maxRetries = 3;
    const retryDelay = 2000;

    try {
      const eventSource = new EventSource(`/api/sessions/${sessionId}/run`);
      eventSourceRef.current = eventSource;

      eventSource.onopen = () => {
        setIsSSEConnected(true);
        setIsInitialLoading(false);
      };

      eventSource.onmessage = (event) => {
        try {
          console.log('SSE message received:', event.data);
          const sseEvent: SSEEvent = JSON.parse(event.data);
          handleSSEEvent(sseEvent);
        } catch (err) {
          console.error('Failed to parse SSE event:', err, 'Raw data:', event.data);
          // Don't set error for parse failures - might be a non-standard event
          // Only set error if it's a critical failure
        }
      };

      eventSource.onerror = (event) => {
        console.error('SSE connection error:', event, 'ReadyState:', eventSource.readyState);
        setIsSSEConnected(false);
        
        // Only show error if connection is actually closed and we've exhausted retries
        if (eventSource.readyState === EventSource.CLOSED) {
          if (retryCount < maxRetries) {
            // Retry connection
            console.log(`Retrying SSE connection (${retryCount + 1}/${maxRetries})`);
            setTimeout(() => {
              connectToSSE(sessionId, retryCount + 1);
            }, retryDelay * (retryCount + 1));
          } else {
            // Only show error if we've exhausted retries and connection is closed
            // Don't show error if we're just waiting for events
            if (!isLoading) {
              setError('Connection lost. Please try again.');
            }
            setIsLoading(false);
            setIsInitialLoading(false);
            eventSource.close();
          }
        } else {
          // Connection is still open (CONNECTING or OPEN) - might be a temporary issue
          console.log('SSE connection state:', eventSource.readyState, '- waiting for events');
        }
      };
    } catch (err) {
      console.error('Failed to create SSE connection:', err);
      setError('Failed to connect to server. Please try again.');
      setIsLoading(false);
      setIsInitialLoading(false);
    }
  };

  const handleSSEEvent = (event: SSEEvent) => {
    const { type, data } = event;

    switch (type) {
      case 'connected':
        // Initial connection event - just acknowledge
        console.log('SSE connected:', data.session_id);
        break;
      
      case 'step_start':
        updateStepStatus(data.step!, 'running', data.timestamp);
        break;
      
      case 'step_complete':
        updateStepStatus(data.step!, 'completed', data.timestamp, data.duration);
        break;
      
      case 'step_error':
        updateStepStatus(data.step!, 'failed', data.timestamp, undefined, data.error);
        break;
      
      case 'session_complete':
        setIsLoading(false);
        console.log('Session complete - artifacts received:', data.artifacts);
        if (data.artifacts) {
          // Ensure proper structure for FinalResult
          const artifacts = data.artifacts as any;
          const finalResult: FinalResult = {
            lesson: artifacts.lesson || null,
            images: artifacts.images || [],
            captions: artifacts.captions || (artifacts.images ? artifacts.images.map((img: any) => img.caption || img.alt_text || '').filter((c: string) => c) : []),
          };
          console.log('Final result structured:', finalResult);
          setFinalResult(finalResult);
        } else {
          console.warn('No artifacts in session_complete event');
        }
        
        // Call session complete endpoint to update BrainPrint
        if (sessionId) {
          const orchestratorURL = getOrchestratorURL();
          fetch(`${orchestratorURL}/api/session/complete`, {
            method: 'POST',
            headers: {
              'Content-Type': 'application/json',
            },
            body: JSON.stringify({
              session_id: sessionId,
              user_id: sessionId, // Use session ID as user ID for now
              explanation_type: explanationType,
            }),
          }).then(() => {
            // Refresh BrainPrint after session completion
            const refreshFn = (window as any)[`brainprint_refresh_${sessionId}`];
            if (refreshFn) {
              refreshFn();
            }
            // Also trigger a manual refresh by updating userID (if needed)
            setTimeout(() => {
              // Force BrainPrint to refresh
              const event = new CustomEvent('brainprint-refresh', { detail: { userID: sessionId } });
              window.dispatchEvent(event);
            }, 500);
          }).catch(err => {
            console.error('Failed to track session completion:', err);
          });
        }
        
        if (eventSourceRef.current) {
          eventSourceRef.current.close();
        }
        break;
      
      case 'session_error':
        setIsLoading(false);
        setError(data.error || 'Session failed');
        if (eventSourceRef.current) {
          eventSourceRef.current.close();
        }
        break;
    }
  };

  const updateStepStatus = (
    stepId: string,
    status: StepStatus['status'],
    timestamp: string,
    duration?: number,
    error?: string
  ) => {
    setSteps(prevSteps =>
      prevSteps.map(step =>
        step.step === stepId
          ? { ...step, status, timestamp, duration, error }
          : step
      )
    );
  };


  // Render content based on explanation type
  const renderContent = () => {
    if (!finalResult || !finalResult.lesson) return null;

    switch (explanationType) {
      case 'visualization':
        return (
          <VisualizationView 
            lesson={finalResult.lesson} 
            topic={topic}
            images={finalResult.images}
          />
        );
      case 'simple':
      case 'analogy':
        return (
          <SimpleExplanationView 
            lesson={finalResult.lesson} 
            topic={topic} 
          />
        );
      default:
        return (
          <LessonCard 
            lesson={finalResult.lesson} 
            images={finalResult.images} 
            sessionId={sessionId || undefined}
            explanationType={explanationType}
            topic={topic}
            onSave={() => {
              // Trigger sidebar refresh if needed
              const event = new CustomEvent('saved-lessons-refresh', { detail: { userID: sessionId } });
              window.dispatchEvent(event);
            }}
          />
        );
    }
  };

  return (
    <>
      <Head>
        <title>ExplainIQ - AI-Powered Learning</title>
        <meta name="description" content="Transform any topic into a structured, visual learning experience" />
        <meta name="viewport" content="width=device-width, initial-scale=1" />
      </Head>

      <div className="min-h-screen bg-gray-50 flex">
        {/* Sidebar */}
        <Sidebar
          activeType={explanationType}
          onTypeChange={setExplanationType}
          disabled={isLoading}
          userID={sessionId || 'default'}
        />

        {/* Main Content */}
        <div className="flex-1 ml-64">
          <div className="container mx-auto px-4 py-8 max-w-5xl">
          {/* Header */}
          <div className="text-center mb-10 animate-fade-in">
            <div className="inline-block mb-4">
              <h1 className="text-5xl font-extrabold bg-gradient-to-r from-blue-600 via-indigo-600 to-purple-600 bg-clip-text text-transparent mb-2">
                ExplainIQ
              </h1>
              <div className="h-1 w-24 bg-gradient-to-r from-blue-500 to-indigo-500 mx-auto rounded-full"></div>
            </div>
            <p className="text-lg text-gray-600 font-medium">
              Transform any topic into a structured, visual learning experience
            </p>
          </div>

          {/* BrainPrint Card */}
          <div className="mb-8">
            <BrainPrintCard 
              userID={sessionId || 'default'} 
              className="mb-6"
              onUpdate={(data) => {
                console.log('BrainPrint updated:', data);
              }}
            />
          </div>

          {/* Topic Form */}
          <div className="bg-white rounded-lg shadow-lg p-6 mb-8 animate-fade-in border border-gray-100">
            <form onSubmit={handleSubmit} className="space-y-4">
              <div>
                <label htmlFor="topic" className="block text-sm font-semibold text-gray-700 mb-2">
                  What would you like to learn about?
                </label>
                <input
                  type="text"
                  id="topic"
                  value={topic}
                  onChange={(e) => {
                    setTopic(e.target.value);
                    setValidationError(null);
                  }}
                  placeholder="e.g., machine learning, quantum computing, blockchain..."
                  className={`
                    w-full px-4 py-3 border rounded-lg transition-all duration-200
                    focus:ring-2 focus:ring-blue-500 focus:border-transparent
                    ${validationError 
                      ? 'border-red-300 focus:ring-red-500' 
                      : 'border-gray-300 focus:ring-blue-500'
                    }
                    ${isLoading ? 'opacity-50 cursor-not-allowed' : ''}
                  `}
                  disabled={isLoading}
                  required
                  aria-invalid={!!validationError}
                  aria-describedby={validationError ? 'topic-error' : undefined}
                />
                {validationError && (
                  <p id="topic-error" className="mt-2 text-sm text-red-600 animate-slide-in" role="alert">
                    {validationError}
                  </p>
                )}
                <p className="mt-2 text-xs text-gray-500">
                  {topic.length}/200 characters
                </p>
              </div>
              <button
                type="submit"
                disabled={isLoading || !topic.trim() || isRetryingSession}
                className={`
                  w-full py-3 px-6 rounded-lg font-semibold 
                  focus:ring-2 focus:ring-offset-2 transition-all duration-200
                  disabled:opacity-50 disabled:cursor-not-allowed
                  flex items-center justify-center space-x-2
                  ${isLoading || isRetryingSession
                    ? 'bg-blue-400 text-white cursor-wait'
                    : 'bg-gradient-to-r from-blue-600 to-indigo-600 text-white hover:from-blue-700 hover:to-indigo-700 shadow-md hover:shadow-lg transform hover:-translate-y-0.5'
                  }
                `}
              >
                {isLoading || isRetryingSession ? (
                  <>
                    <svg className="animate-spin h-5 w-5" fill="none" viewBox="0 0 24 24">
                      <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
                      <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                    </svg>
                    <span>Creating Learning Experience...</span>
                  </>
                ) : (
                  <>
                    <svg className="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 10V3L4 14h7v7l9-11h-7z" />
                    </svg>
                    <span>Start Learning</span>
                  </>
                )}
              </button>
            </form>
          </div>

          {/* Error Display */}
          {error && (
            <ErrorMessage
              title="Error"
              message={error}
              onRetry={() => {
                setError(null);
                if (sessionId) {
                  connectToSSE(sessionId);
                } else {
                  handleSubmit(new Event('submit') as any);
                }
              }}
              onDismiss={() => setError(null)}
              type="error"
            />
          )}

          {/* Connection Status */}
          {isLoading && !isSSEConnected && !error && (
            <div className="bg-blue-50 border border-blue-200 rounded-lg p-4 mb-8 animate-pulse-soft">
              <div className="flex items-center space-x-3">
                <svg className="animate-spin h-5 w-5 text-blue-600" fill="none" viewBox="0 0 24 24">
                  <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
                  <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                </svg>
                <p className="text-sm text-blue-800">Connecting to server...</p>
              </div>
            </div>
          )}

          {/* Progress Timeline */}
          {sessionId && (
            <div className="bg-white rounded-lg shadow-lg p-6 mb-8 animate-fade-in border border-gray-100">
              <div className="flex items-center justify-between mb-6">
                <h2 className="text-xl font-semibold text-gray-900">Learning Progress</h2>
                {isSSEConnected && (
                  <div className="flex items-center space-x-2 text-sm text-green-600">
                    <div className="w-2 h-2 bg-green-500 rounded-full animate-pulse"></div>
                    <span>Connected</span>
                  </div>
                )}
              </div>
              {isInitialLoading ? (
                <TimelineSkeleton />
              ) : (
                <Timeline steps={steps} stepInfo={STEPS} />
              )}
            </div>
          )}

          {/* Final Result */}
          {finalResult && (
            <ErrorBoundary>
              <div className="bg-white rounded-lg shadow-lg p-6 animate-fade-in border border-gray-100">
                <div className="flex items-center justify-between mb-6">
                  <h2 className="text-2xl font-bold text-gray-900 bg-gradient-to-r from-blue-600 to-indigo-600 bg-clip-text text-transparent">
                    Your Learning Experience
                  </h2>
                  {explanationType !== 'standard' && (
                    <button
                      onClick={() => setExplanationType('standard')}
                      className="text-sm text-blue-600 hover:text-blue-700 font-medium transition-colors flex items-center space-x-1"
                    >
                      <svg className="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 12h16M4 18h16" />
                      </svg>
                      <span>View Standard Format</span>
                    </button>
                  )}
                </div>
                {renderContent()}
              </div>
            </ErrorBoundary>
          )}
          </div>
        </div>
      </div>
    </>
  );
}

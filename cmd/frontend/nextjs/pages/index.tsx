import { useState, useRef, useEffect } from 'react';
import Head from 'next/head';
import { SessionRequest, SessionResponse, SSEEvent, StepStatus, FinalResult } from '../types';
import Timeline from '../components/Timeline';
import LessonCard from '../components/LessonCard';

const STEPS = [
  { id: 'summarizer', name: 'Summarizer', description: 'Analyzing topic and context' },
  { id: 'explainer', name: 'Explainer', description: 'Creating structured lesson' },
  { id: 'visualizer', name: 'Visualizer', description: 'Generating diagrams' },
  { id: 'critic', name: 'Critic', description: 'Reviewing and improving' },
];

export default function Home() {
  const [topic, setTopic] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [sessionId, setSessionId] = useState<string | null>(null);
  const [steps, setSteps] = useState<StepStatus[]>([]);
  const [finalResult, setFinalResult] = useState<FinalResult | null>(null);
  const [error, setError] = useState<string | null>(null);
  const eventSourceRef = useRef<EventSource | null>(null);

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
    if (!topic.trim()) return;

    setIsLoading(true);
    setError(null);
    setFinalResult(null);
    setSessionId(null);

    try {
      // Create session
      const sessionRequest: SessionRequest = { topic: topic.trim() };
      const response = await fetch('/api/sessions', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(sessionRequest),
      });

      if (!response.ok) {
        throw new Error(`Failed to create session: ${response.statusText}`);
      }

      const sessionData: SessionResponse = await response.json();
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
      setError(err instanceof Error ? err.message : 'An error occurred');
      setIsLoading(false);
    }
  };

  const connectToSSE = (sessionId: string) => {
    if (eventSourceRef.current) {
      eventSourceRef.current.close();
    }

    const eventSource = new EventSource(`/api/sessions/${sessionId}/run`);
    eventSourceRef.current = eventSource;

    eventSource.onmessage = (event) => {
      try {
        const sseEvent: SSEEvent = JSON.parse(event.data);
        handleSSEEvent(sseEvent);
      } catch (err) {
        console.error('Failed to parse SSE event:', err);
      }
    };

    eventSource.onerror = (event) => {
      console.error('SSE connection error:', event);
      setError('Connection lost. Please try again.');
      setIsLoading(false);
      eventSource.close();
    };
  };

  const handleSSEEvent = (event: SSEEvent) => {
    const { type, data } = event;

    switch (type) {
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
        if (data.artifacts) {
          setFinalResult(data.artifacts as FinalResult);
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


  return (
    <>
      <Head>
        <title>ExplainIQ - AI-Powered Learning</title>
        <meta name="description" content="Transform any topic into a structured, visual learning experience" />
        <meta name="viewport" content="width=device-width, initial-scale=1" />
      </Head>

      <div className="min-h-screen bg-gray-50">
        <div className="container mx-auto px-4 py-8 max-w-4xl">
          {/* Header */}
          <div className="text-center mb-8">
            <h1 className="text-4xl font-bold text-gray-900 mb-2">ExplainIQ</h1>
            <p className="text-lg text-gray-600">Transform any topic into a structured, visual learning experience</p>
          </div>

          {/* Topic Form */}
          <div className="bg-white rounded-lg shadow-md p-6 mb-8">
            <form onSubmit={handleSubmit} className="space-y-4">
              <div>
                <label htmlFor="topic" className="block text-sm font-medium text-gray-700 mb-2">
                  What would you like to learn about?
                </label>
                <input
                  type="text"
                  id="topic"
                  value={topic}
                  onChange={(e) => setTopic(e.target.value)}
                  placeholder="e.g., machine learning, quantum computing, blockchain..."
                  className="w-full px-4 py-3 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                  disabled={isLoading}
                  required
                />
              </div>
              <button
                type="submit"
                disabled={isLoading || !topic.trim()}
                className="w-full bg-blue-600 text-white py-3 px-6 rounded-lg font-medium hover:bg-blue-700 focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
              >
                {isLoading ? 'Creating Learning Experience...' : 'Start Learning'}
              </button>
            </form>
          </div>

          {/* Error Display */}
          {error && (
            <div className="bg-red-50 border border-red-200 rounded-lg p-4 mb-8">
              <div className="flex">
                <div className="flex-shrink-0">
                  <span className="text-red-400">❌</span>
                </div>
                <div className="ml-3">
                  <h3 className="text-sm font-medium text-red-800">Error</h3>
                  <p className="text-sm text-red-700 mt-1">{error}</p>
                </div>
              </div>
            </div>
          )}

          {/* Progress Timeline */}
          {sessionId && (
            <div className="bg-white rounded-lg shadow-md p-6 mb-8">
              <h2 className="text-xl font-semibold text-gray-900 mb-6">Learning Progress</h2>
              <Timeline steps={steps} stepInfo={STEPS} />
            </div>
          )}

          {/* Final Result */}
          {finalResult && (
            <div className="bg-white rounded-lg shadow-md p-6">
              <h2 className="text-2xl font-semibold text-gray-900 mb-6">Your Learning Experience</h2>
              <LessonCard lesson={finalResult.lesson} images={finalResult.images} sessionId={sessionId || undefined} />
            </div>
          )}
        </div>
      </div>
    </>
  );
}

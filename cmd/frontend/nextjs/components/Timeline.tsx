import React, { useState, useEffect } from 'react';
import { StepStatus } from '../types';

interface TimelineProps {
  steps: StepStatus[];
  stepInfo: Array<{ id: string; name: string; description: string }>;
}

const STEP_MESSAGES: Record<string, string> = {
  summarizer: 'Summarizer complete — context analyzed!',
  explainer: 'Explainer complete — structured clarity achieved!',
  visualizer: 'Visualizer ready — your mind map is live!',
  critic: 'Critic complete — lesson refined and improved!',
};

const STEP_ICONS: Record<string, string> = {
  summarizer: '📊',
  explainer: '📚',
  visualizer: '🎨',
  critic: '✨',
};

const Timeline: React.FC<TimelineProps> = ({ steps, stepInfo }) => {
  const [completedSteps, setCompletedSteps] = useState<Set<string>>(new Set());
  const [showMessages, setShowMessages] = useState<Set<string>>(new Set());

  useEffect(() => {
    steps.forEach(step => {
      if (step.status === 'completed' && !completedSteps.has(step.step)) {
        setCompletedSteps(prev => new Set(prev).add(step.step));
        // Show success message
        setShowMessages(prev => new Set(prev).add(step.step));
        // Hide message after 3 seconds
        setTimeout(() => {
          setShowMessages(prev => {
            const next = new Set(prev);
            next.delete(step.step);
            return next;
          });
        }, 3000);
      }
    });
  }, [steps, completedSteps]);

  const getStepIcon = (status: StepStatus['status'], stepId: string) => {
    const icon = STEP_ICONS[stepId] || '⏳';
    switch (status) {
      case 'pending':
        return '⏳';
      case 'running':
        return '🔄';
      case 'completed':
        return '✅';
      case 'failed':
        return '❌';
      default:
        return icon;
    }
  };

  const getStepColor = (status: StepStatus['status']) => {
    switch (status) {
      case 'pending':
        return 'text-gray-500 bg-gray-100';
      case 'running':
        return 'text-blue-600 bg-blue-100 animate-pulse';
      case 'completed':
        return 'text-green-600 bg-green-100';
      case 'failed':
        return 'text-red-600 bg-red-100';
      default:
        return 'text-gray-500 bg-gray-100';
    }
  };

  const getProgressPercentage = (stepIndex: number) => {
    const completedCount = steps.filter(s => s.status === 'completed').length;
    return Math.round((completedCount / steps.length) * 100);
  };

  const overallProgress = getProgressPercentage(0);

  return (
    <div className="space-y-6">
      {/* Overall Progress Bar */}
      <div className="bg-white rounded-lg shadow-md p-4">
        <div className="flex justify-between items-center mb-2">
          <h3 className="text-sm font-semibold text-gray-700">Learning Progress</h3>
          <span className="text-sm font-bold text-blue-600">{overallProgress}%</span>
        </div>
        <div className="w-full bg-gray-200 rounded-full h-3 overflow-hidden">
          <div
            className="h-3 rounded-full bg-gradient-to-r from-blue-500 via-indigo-500 to-purple-500 transition-all duration-500 ease-out shadow-sm"
            style={{ width: `${overallProgress}%` }}
          >
            <div className="h-full w-full bg-gradient-to-r from-transparent via-white/20 to-transparent animate-shimmer"></div>
          </div>
        </div>
      </div>

      {/* Steps Timeline */}
      {steps.map((step, index) => {
        const stepInfoItem = stepInfo.find(s => s.id === step.step);
        const isCompleted = step.status === 'completed';
        const isRunning = step.status === 'running';
        const showMessage = showMessages.has(step.step);
        const stepProgress = isCompleted ? 100 : isRunning ? 50 : 0;

        return (
          <div
            key={step.step}
            className={`flex items-start space-x-4 transition-all duration-300 ${
              isCompleted ? 'transform scale-[1.02]' : ''
            }`}
          >
            {/* Timeline Line and Icon */}
            <div className="flex flex-col items-center">
              <div
                className={`w-12 h-12 rounded-full flex items-center justify-center text-lg font-medium transition-all duration-300 shadow-md ${
                  isCompleted ? 'animate-pulse-once' : ''
                } ${getStepColor(step.status)}`}
              >
                {getStepIcon(step.status, step.step)}
              </div>
              {index < steps.length - 1 && (
                <div
                  className={`w-0.5 h-16 mt-2 transition-all duration-500 ${
                    isCompleted ? 'bg-gradient-to-b from-green-500 to-blue-200' : 'bg-gray-200'
                  }`}
                ></div>
              )}
            </div>

            {/* Step Content */}
            <div className="flex-1 min-w-0 bg-white rounded-lg shadow-md p-4 transition-all duration-300 hover:shadow-lg">
              <div className="flex items-center justify-between mb-2">
                <h3 className="text-lg font-medium text-gray-900 flex items-center space-x-2">
                  <span>{stepInfoItem?.name}</span>
                  {isCompleted && (
                    <span className="text-xs bg-green-100 text-green-700 px-2 py-1 rounded-full font-semibold animate-fade-in">
                      Complete
                    </span>
                  )}
                  {isRunning && (
                    <span className="text-xs bg-blue-100 text-blue-700 px-2 py-1 rounded-full font-semibold animate-pulse">
                      Running...
                    </span>
                  )}
                </h3>
                {step.duration && (
                  <span className="text-sm text-gray-500">
                    {Math.round(step.duration)}ms
                  </span>
                )}
              </div>
              
              <p className="text-sm text-gray-600 mb-3">
                {stepInfoItem?.description}
              </p>

              {/* Progress Bar for Individual Step */}
              <div className="mb-2">
                <div className="w-full bg-gray-200 rounded-full h-2 overflow-hidden">
                  <div
                    className={`h-2 rounded-full transition-all duration-500 ease-out ${
                      isCompleted
                        ? 'bg-gradient-to-r from-green-500 to-emerald-500'
                        : isRunning
                        ? 'bg-gradient-to-r from-blue-500 to-indigo-500 animate-pulse'
                        : 'bg-gray-300'
                    }`}
                    style={{ width: `${stepProgress}%` }}
                  >
                    {isRunning && (
                      <div className="h-full w-full bg-gradient-to-r from-transparent via-white/20 to-transparent animate-shimmer"></div>
                    )}
                  </div>
                </div>
              </div>

              {/* Success Message */}
              {showMessage && STEP_MESSAGES[step.step] && (
                <div className="mt-3 p-3 bg-green-50 border border-green-200 rounded-lg animate-fade-in">
                  <p className="text-sm text-green-800 font-medium flex items-center space-x-2">
                    <span>✨</span>
                    <span>{STEP_MESSAGES[step.step]}</span>
                  </p>
                </div>
              )}

              {step.error && (
                <p className="text-sm text-red-600 mt-2 bg-red-50 p-2 rounded">
                  Error: {step.error}
                </p>
              )}
              
              <p className="text-xs text-gray-400 mt-2">
                {new Date(step.timestamp).toLocaleTimeString()}
              </p>
            </div>
          </div>
        );
      })}
    </div>
  );
};

export default Timeline;

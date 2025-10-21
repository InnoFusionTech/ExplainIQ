import React from 'react';
import { StepStatus } from '../types';

interface TimelineProps {
  steps: StepStatus[];
  stepInfo: Array<{ id: string; name: string; description: string }>;
}

const Timeline: React.FC<TimelineProps> = ({ steps, stepInfo }) => {
  const getStepIcon = (status: StepStatus['status']) => {
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
        return '⏳';
    }
  };

  const getStepColor = (status: StepStatus['status']) => {
    switch (status) {
      case 'pending':
        return 'text-gray-500';
      case 'running':
        return 'text-blue-600';
      case 'completed':
        return 'text-green-600';
      case 'failed':
        return 'text-red-600';
      default:
        return 'text-gray-500';
    }
  };

  return (
    <div className="space-y-6">
      {steps.map((step, index) => {
        const stepInfoItem = stepInfo.find(s => s.id === step.step);
        return (
          <div key={step.step} className="flex items-start space-x-4">
            {/* Timeline Line */}
            <div className="flex flex-col items-center">
              <div className={`w-8 h-8 rounded-full flex items-center justify-center text-sm font-medium ${getStepColor(step.status)}`}>
                {getStepIcon(step.status)}
              </div>
              {index < steps.length - 1 && (
                <div className="w-0.5 h-16 bg-gray-200 mt-2"></div>
              )}
            </div>

            {/* Step Content */}
            <div className="flex-1 min-w-0">
              <div className="flex items-center justify-between">
                <h3 className="text-lg font-medium text-gray-900">
                  {stepInfoItem?.name}
                </h3>
                {step.duration && (
                  <span className="text-sm text-gray-500">
                    {step.duration}ms
                  </span>
                )}
              </div>
              <p className="text-sm text-gray-600 mt-1">
                {stepInfoItem?.description}
              </p>
              {step.error && (
                <p className="text-sm text-red-600 mt-2">
                  Error: {step.error}
                </p>
              )}
              <p className="text-xs text-gray-400 mt-1">
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




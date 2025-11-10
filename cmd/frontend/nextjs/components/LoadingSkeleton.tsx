import React from 'react';

interface SkeletonProps {
  className?: string;
}

export const Skeleton: React.FC<SkeletonProps> = ({ className = '' }) => {
  return (
    <div className={`animate-pulse bg-gray-200 rounded ${className}`} />
  );
};

export const LoadingSkeleton: React.FC = () => {
  return (
    <div className="space-y-6">
      <div className="bg-white rounded-lg shadow-md p-6">
        <Skeleton className="h-8 w-3/4 mb-4" />
        <Skeleton className="h-4 w-full mb-2" />
        <Skeleton className="h-4 w-5/6 mb-2" />
        <Skeleton className="h-4 w-4/6" />
      </div>
      
      <div className="bg-white rounded-lg shadow-md p-6">
        <Skeleton className="h-6 w-1/2 mb-4" />
        <div className="space-y-3">
          {[1, 2, 3, 4].map((i) => (
            <div key={i} className="flex items-center space-x-4">
              <Skeleton className="w-8 h-8 rounded-full" />
              <div className="flex-1">
                <Skeleton className="h-4 w-1/3 mb-2" />
                <Skeleton className="h-3 w-2/3" />
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
};

export const TimelineSkeleton: React.FC = () => {
  return (
    <div className="space-y-6">
      {[1, 2, 3, 4].map((i) => (
        <div key={i} className="flex items-start space-x-4">
          <div className="flex flex-col items-center">
            <Skeleton className="w-8 h-8 rounded-full" />
            {i < 4 && <Skeleton className="w-0.5 h-16 mt-2" />}
          </div>
          <div className="flex-1">
            <Skeleton className="h-5 w-1/4 mb-2" />
            <Skeleton className="h-4 w-3/4 mb-1" />
            <Skeleton className="h-3 w-1/6" />
          </div>
        </div>
      ))}
    </div>
  );
};

export default LoadingSkeleton;













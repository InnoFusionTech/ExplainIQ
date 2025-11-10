import React, { useState } from 'react';

interface RetryButtonProps {
  onRetry: () => Promise<void>;
  maxRetries?: number;
  retryDelay?: number;
  className?: string;
}

const RetryButton: React.FC<RetryButtonProps> = ({
  onRetry,
  maxRetries = 3,
  retryDelay = 1000,
  className = '',
}) => {
  const [isRetrying, setIsRetrying] = useState(false);
  const [retryCount, setRetryCount] = useState(0);
  const [lastError, setLastError] = useState<string | null>(null);

  const handleRetry = async () => {
    if (retryCount >= maxRetries) {
      setLastError(`Maximum retries (${maxRetries}) exceeded`);
      return;
    }

    setIsRetrying(true);
    setLastError(null);

    try {
      await new Promise(resolve => setTimeout(resolve, retryDelay * (retryCount + 1)));
      await onRetry();
      setRetryCount(0);
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : 'Retry failed';
      setLastError(errorMessage);
      setRetryCount(prev => prev + 1);
    } finally {
      setIsRetrying(false);
    }
  };

  if (retryCount >= maxRetries) {
    return (
      <div className={`text-center ${className}`}>
        <p className="text-sm text-red-600 mb-2">{lastError}</p>
        <button
          onClick={() => {
            setRetryCount(0);
            setLastError(null);
          }}
          className="text-blue-600 hover:text-blue-700 text-sm font-medium"
        >
          Reset and try again
        </button>
      </div>
    );
  }

  return (
    <button
      onClick={handleRetry}
      disabled={isRetrying}
      className={`
        ${className}
        bg-blue-600 text-white px-4 py-2 rounded-lg font-medium
        hover:bg-blue-700 focus:ring-2 focus:ring-blue-500 focus:ring-offset-2
        disabled:opacity-50 disabled:cursor-not-allowed
        transition-colors flex items-center space-x-2
      `}
    >
      {isRetrying ? (
        <>
          <svg className="animate-spin h-4 w-4" fill="none" viewBox="0 0 24 24">
            <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
            <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
          </svg>
          <span>Retrying... ({retryCount + 1}/{maxRetries})</span>
        </>
      ) : (
        <>
          <svg className="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
          </svg>
          <span>Retry</span>
        </>
      )}
    </button>
  );
};

export default RetryButton;












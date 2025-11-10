import { useState, useCallback } from 'react';

interface UseRetryOptions {
  maxRetries?: number;
  retryDelay?: number;
  onSuccess?: () => void;
  onError?: (error: Error) => void;
}

export const useRetry = <T extends (...args: any[]) => Promise<any>>(
  fn: T,
  options: UseRetryOptions = {}
) => {
  const {
    maxRetries = 3,
    retryDelay = 1000,
    onSuccess,
    onError,
  } = options;

  const [isRetrying, setIsRetrying] = useState(false);
  const [retryCount, setRetryCount] = useState(0);
  const [error, setError] = useState<Error | null>(null);

  const execute = useCallback(
    async (...args: Parameters<T>): Promise<ReturnType<T> | null> => {
      setIsRetrying(true);
      setError(null);

      for (let attempt = 0; attempt <= maxRetries; attempt++) {
        try {
          const result = await fn(...args);
          setRetryCount(0);
          setIsRetrying(false);
          onSuccess?.();
          return result;
        } catch (err) {
          const error = err instanceof Error ? err : new Error('Unknown error');
          setError(error);

          if (attempt < maxRetries) {
            await new Promise(resolve => setTimeout(resolve, retryDelay * (attempt + 1)));
            setRetryCount(attempt + 1);
          } else {
            setIsRetrying(false);
            onError?.(error);
          }
        }
      }

      return null;
    },
    [fn, maxRetries, retryDelay, onSuccess, onError]
  );

  const reset = useCallback(() => {
    setRetryCount(0);
    setError(null);
    setIsRetrying(false);
  }, []);

  return {
    execute,
    isRetrying,
    retryCount,
    error,
    reset,
    hasExceededMaxRetries: retryCount >= maxRetries,
  };
};













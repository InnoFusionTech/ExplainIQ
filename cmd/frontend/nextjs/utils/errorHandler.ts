export class AppError extends Error {
  constructor(
    message: string,
    public code?: string,
    public statusCode?: number,
    public retryable: boolean = false
  ) {
    super(message);
    this.name = 'AppError';
  }
}

export const handleError = (error: unknown): { message: string; retryable: boolean } => {
  if (error instanceof AppError) {
    return {
      message: error.message,
      retryable: error.retryable,
    };
  }
  
  if (error instanceof Error) {
    // Network errors are usually retryable
    if (error.message.includes('fetch') || error.message.includes('network')) {
      return {
        message: 'Network error. Please check your connection and try again.',
        retryable: true,
      };
    }
    
    return {
      message: error.message,
      retryable: false,
    };
  }
  
  return {
    message: 'An unexpected error occurred. Please try again.',
    retryable: true,
  };
};

export const isRetryableError = (error: unknown): boolean => {
  if (error instanceof AppError) {
    return error.retryable;
  }
  
  if (error instanceof Error) {
    // Common retryable error patterns
    const retryablePatterns = [
      /network/i,
      /timeout/i,
      /connection/i,
      /502/i,
      /503/i,
      /504/i,
    ];
    
    return retryablePatterns.some(pattern => pattern.test(error.message));
  }
  
  return false;
};






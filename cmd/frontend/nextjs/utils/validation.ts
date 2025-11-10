export interface ValidationResult {
  valid: boolean;
  error?: string;
}

export const validateTopic = (topic: string): ValidationResult => {
  const trimmed = topic.trim();
  
  if (!trimmed) {
    return {
      valid: false,
      error: 'Please enter a topic',
    };
  }
  
  if (trimmed.length < 3) {
    return {
      valid: false,
      error: 'Topic must be at least 3 characters long',
    };
  }
  
  if (trimmed.length > 200) {
    return {
      valid: false,
      error: 'Topic must be less than 200 characters',
    };
  }
  
  // Check for suspicious patterns (optional security measure)
  const suspiciousPatterns = [
    /<script/i,
    /javascript:/i,
    /on\w+\s*=/i,
  ];
  
  for (const pattern of suspiciousPatterns) {
    if (pattern.test(trimmed)) {
      return {
        valid: false,
        error: 'Topic contains invalid characters',
      };
    }
  }
  
  return { valid: true };
};













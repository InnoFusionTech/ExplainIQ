// Types for the ExplainIQ Frontend

export interface SessionRequest {
  topic: string;
}

export interface SessionResponse {
  session_id: string;
  status: string;
}

export interface StepStatus {
  step: string;
  status: 'pending' | 'running' | 'completed' | 'failed';
  timestamp: string;
  duration?: number;
  error?: string;
}

export interface SSEEvent {
  type: 'step_start' | 'step_complete' | 'step_error' | 'session_complete' | 'session_error';
  data: {
    session_id: string;
    step?: string;
    status?: string;
    timestamp: string;
    duration?: number;
    error?: string;
    artifacts?: Record<string, any>;
  };
}

export interface OGLesson {
  big_picture: string;
  metaphor: string;
  core_mechanism: string;
  toy_example_code: string;
  memory_hook: string;
  real_life: string;
  best_practices: string;
}

export interface ImageRef {
  url: string;
  alt_text: string;
  caption: string;
}

export interface FinalResult {
  lesson: OGLesson;
  images: ImageRef[];
  captions: string[];
}

export interface PDFResponse {
  pdf_url: string;
  filename: string;
  size: number;
  created_at: string;
}

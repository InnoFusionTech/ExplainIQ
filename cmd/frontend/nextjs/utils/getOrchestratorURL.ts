/**
 * Get the orchestrator URL for client-side requests
 * This function handles both build-time and runtime environment variables
 */
export function getOrchestratorURL(): string {
  // In browser/client-side code, check for window object
  if (typeof window !== 'undefined') {
    // Priority 1: Check if we're on Cloud Run first (before checking window variable)
    // This ensures we use the correct URL even if _app.tsx hasn't set it yet
    const currentHost = window.location.hostname;
    if (currentHost.includes('.a.run.app') || currentHost.includes('.run.app')) {
      // Cloud Run domain - use known orchestrator URL
      // Known orchestrator URL from deployment: https://explainiq-orchestrator-othekugkka-ew.a.run.app
      // This is set as a fallback since Cloud Run service hashes are different
      return 'https://explainiq-orchestrator-othekugkka-ew.a.run.app';
    }
    
    // Priority 2: Try to get from window object (set at runtime via _app.tsx)
    const windowURL = (window as any).__ORCHESTRATOR_URL__;
    if (windowURL && windowURL !== 'http://localhost:8080' && !windowURL.includes('localhost')) {
      return windowURL;
    }
    
    // Priority 3: Try to get from environment variable (available if set at build time)
    // Note: In client-side code, NEXT_PUBLIC_* vars are embedded at build time
    const envURL = process.env.NEXT_PUBLIC_ORCHESTRATOR_URL;
    if (envURL && envURL !== 'http://localhost:8080' && !envURL.includes('localhost')) {
      return envURL;
    }
    
    // Priority 4: Check if we're on localhost (development)
    if (currentHost === 'localhost' || currentHost === '127.0.0.1') {
      return 'http://localhost:8080';
    }
  }
  
  // Server-side or fallback
  const serverURL = process.env.NEXT_PUBLIC_ORCHESTRATOR_URL || 
                    process.env.ORCHESTRATOR_URL;
  if (serverURL && serverURL !== 'http://localhost:8080' && !serverURL.includes('localhost')) {
    return serverURL;
  }
  
  // Final fallback
  return 'http://localhost:8080';
}



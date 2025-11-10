/**
 * Get the orchestrator URL for client-side requests
 * This function handles both build-time and runtime environment variables
 */
export function getOrchestratorURL(): string {
  // In browser/client-side code, check for window object
  if (typeof window !== 'undefined') {
    // Try to get from window object (set at runtime)
    const windowURL = (window as any).__ORCHESTRATOR_URL__;
    if (windowURL) {
      return windowURL;
    }
    
    // Try to get from environment variable (available if set at build time)
    const envURL = process.env.NEXT_PUBLIC_ORCHESTRATOR_URL;
    if (envURL && envURL !== 'http://localhost:8080') {
      return envURL;
    }
    
    // Fallback: construct from current window location
    // If frontend is on Cloud Run, orchestrator should be on same domain pattern
    const currentHost = window.location.hostname;
    if (currentHost.includes('.a.run.app')) {
      // Extract the service name and construct orchestrator URL
      // Frontend: explainiq-frontend-xxx-ew.a.run.app
      // Orchestrator: explainiq-orchestrator-xxx-ew.a.run.app
      const orchestratorHost = currentHost.replace('explainiq-frontend', 'explainiq-orchestrator');
      return `https://${orchestratorHost}`;
    }
  }
  
  // Server-side or fallback
  return process.env.NEXT_PUBLIC_ORCHESTRATOR_URL || 
         process.env.ORCHESTRATOR_URL || 
         'http://localhost:8080';
}


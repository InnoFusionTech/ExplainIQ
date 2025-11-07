/**
 * Get the orchestrator URL from environment variables
 * Logs the URL being used for debugging
 */
export function getOrchestratorURL(): string {
  const url = process.env.ORCHESTRATOR_URL || 'http://localhost:8080';
  
  // Log the URL in production for debugging (remove sensitive info)
  if (process.env.NODE_ENV === 'production') {
    console.log(`[Orchestrator] Using URL: ${url}`);
  }
  
  return url;
}






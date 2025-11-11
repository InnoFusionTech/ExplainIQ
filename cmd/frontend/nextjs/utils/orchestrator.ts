/**
 * Get the orchestrator URL from environment variables
 * This is for server-side use (API routes, server components)
 * For client-side, use getOrchestratorURL from utils/getOrchestratorURL.ts
 */
export function getOrchestratorURL(): string {
  // Priority 1: Check environment variables (set at runtime in Cloud Run)
  let url = process.env.ORCHESTRATOR_URL || process.env.NEXT_PUBLIC_ORCHESTRATOR_URL;
  
  // Priority 2: If on Cloud Run but env var not set, use known orchestrator URL
  // Check if we're running in Cloud Run by checking for Cloud Run environment variables
  if ((!url || url.includes('localhost')) && process.env.K_SERVICE) {
    // We're in Cloud Run, use known orchestrator URL
    url = 'https://explainiq-orchestrator-othekugkka-ew.a.run.app';
  }
  
  // Priority 3: Fallback to localhost for development
  if (!url || url.includes('localhost')) {
    url = 'http://localhost:8080';
  }
  
  // Log the URL in production for debugging
  if (process.env.NODE_ENV === 'production') {
    console.log(`[Orchestrator] Using URL: ${url}`);
  }
  
  return url;
}













import type { AppProps } from 'next/app';
import { useEffect } from 'react';
import ErrorBoundary from '../components/ErrorBoundary';
import '../styles/globals.css';

export default function App({ Component, pageProps }: AppProps) {
  useEffect(() => {
    // Set orchestrator URL at runtime for client-side code
    // Fetch from server-side API route which has access to runtime environment variables
    if (typeof window !== 'undefined' && !(window as any).__ORCHESTRATOR_URL__) {
      fetch('/api/orchestrator-url')
        .then(res => res.json())
        .then(data => {
          if (data.url && data.url !== 'http://localhost:8080' && !data.url.includes('localhost')) {
            (window as any).__ORCHESTRATOR_URL__ = data.url;
          } else if (window.location.hostname.includes('.run.app') || window.location.hostname.includes('.a.run.app')) {
            // Fallback: use known orchestrator URL if on Cloud Run
            (window as any).__ORCHESTRATOR_URL__ = 'https://explainiq-orchestrator-othekugkka-ew.a.run.app';
          }
        })
        .catch(err => {
          console.error('Failed to fetch orchestrator URL:', err);
          // Fallback: use known orchestrator URL if on Cloud Run
          if (window.location.hostname.includes('.run.app') || window.location.hostname.includes('.a.run.app')) {
            (window as any).__ORCHESTRATOR_URL__ = 'https://explainiq-orchestrator-othekugkka-ew.a.run.app';
          }
        });
    }
  }, []);

  return (
    <ErrorBoundary>
      <Component {...pageProps} />
    </ErrorBoundary>
  );
}




import { NextApiRequest, NextApiResponse } from 'next';
import { SSEEvent } from '../../../../types';

const ORCHESTRATOR_URL = process.env.ORCHESTRATOR_URL || 'http://localhost:8080';

export default async function handler(
  req: NextApiRequest,
  res: NextApiResponse
) {
  if (req.method !== 'GET') {
    return res.status(405).json({ error: 'Method not allowed' });
  }

  const { id: sessionId } = req.query;

  if (!sessionId || typeof sessionId !== 'string') {
    return res.status(400).json({ error: 'Session ID is required' });
  }

  // Set up SSE headers
  res.writeHead(200, {
    'Content-Type': 'text/event-stream',
    'Cache-Control': 'no-cache',
    'Connection': 'keep-alive',
    'Access-Control-Allow-Origin': '*',
    'Access-Control-Allow-Headers': 'Cache-Control',
  });

  // Send initial connection event
  const initialEvent: SSEEvent = {
    type: 'step_start',
    data: {
      session_id: sessionId,
      step: 'connection',
      status: 'connected',
      timestamp: new Date().toISOString(),
    },
  };
  res.write(`data: ${JSON.stringify(initialEvent)}\n\n`);

  try {
    // Connect to orchestrator SSE
    const orchestratorSSE = new EventSource(`${ORCHESTRATOR_URL}/api/sessions/run?id=${sessionId}`);

    orchestratorSSE.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data);
        
        // Forward the event to the client
        res.write(`data: ${JSON.stringify(data)}\n\n`);
      } catch (error) {
        console.error('Error parsing SSE event:', error);
      }
    };

    orchestratorSSE.onerror = (error) => {
      console.error('Orchestrator SSE error:', error);
      
      const errorEvent: SSEEvent = {
        type: 'session_error',
        data: {
          session_id: sessionId,
          status: 'error',
          timestamp: new Date().toISOString(),
          error: 'Connection to orchestrator lost',
        },
      };
      res.write(`data: ${JSON.stringify(errorEvent)}\n\n`);
      
      orchestratorSSE.close();
      res.end();
    };

    // Handle client disconnect
    req.on('close', () => {
      orchestratorSSE.close();
      res.end();
    });

  } catch (error) {
    console.error('SSE setup error:', error);
    
    const errorEvent: SSEEvent = {
      type: 'session_error',
      data: {
        session_id: sessionId,
        status: 'error',
        timestamp: new Date().toISOString(),
        error: error instanceof Error ? error.message : 'Unknown error',
      },
    };
    res.write(`data: ${JSON.stringify(errorEvent)}\n\n`);
    res.end();
  }
}

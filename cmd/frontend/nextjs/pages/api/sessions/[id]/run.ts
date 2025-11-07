import { NextApiRequest, NextApiResponse } from 'next';
import { SSEEvent } from '../../../../types';
import { getOrchestratorURL } from '../../../../utils/orchestrator';

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
    // Connect to orchestrator SSE using fetch (Node.js 18+)
    // Note: Orchestrator expects POST /api/sessions/{id}/run
    const ORCHESTRATOR_URL = getOrchestratorURL();
    const orchestratorEndpoint = `${ORCHESTRATOR_URL}/api/sessions/${sessionId}/run`;
    console.log(`[SSE Connection] Connecting to: ${orchestratorEndpoint}`);
    console.log(`[SSE Connection] Environment check - ORCHESTRATOR_URL from env: ${process.env.ORCHESTRATOR_URL || 'NOT SET'}`);
    
    const response = await fetch(orchestratorEndpoint, {
      method: 'POST',
      headers: {
        'Accept': 'text/event-stream',
        'Cache-Control': 'no-cache',
      },
    });

    if (!response.ok) {
      throw new Error(`Orchestrator SSE error: ${response.status} ${response.statusText}`);
    }

    if (!response.body) {
      throw new Error('Response body is null');
    }

    // Handle client disconnect
    req.on('close', () => {
      // Response body will be closed automatically when client disconnects
      res.end();
    });

    // Stream the SSE response from orchestrator to client
    const reader = response.body.getReader();
    const decoder = new TextDecoder();
    let buffer = '';

    try {
      while (true) {
        const { done, value } = await reader.read();

        if (done) {
          break;
        }

        // Decode the chunk and add to buffer
        buffer += decoder.decode(value, { stream: true });

        // Process complete SSE messages (ending with \n\n)
        const lines = buffer.split('\n\n');
        buffer = lines.pop() || ''; // Keep incomplete message in buffer

        for (const line of lines) {
          if (line.trim()) {
            // Forward the SSE event to the client
            res.write(`${line}\n\n`);
            
            // Flush to send immediately
            if (typeof (res as any).flush === 'function') {
              (res as any).flush();
            }
          }
        }
      }

      // Send any remaining buffered data
      if (buffer.trim()) {
        res.write(`${buffer}\n\n`);
      }
    } catch (streamError) {
      console.error('Error streaming SSE:', streamError);
      const errorEvent: SSEEvent = {
        type: 'session_error',
        data: {
          session_id: sessionId,
          status: 'error',
          timestamp: new Date().toISOString(),
          error: streamError instanceof Error ? streamError.message : 'Stream error',
        },
      };
      res.write(`data: ${JSON.stringify(errorEvent)}\n\n`);
    } finally {
      reader.releaseLock();
      if (!res.writableEnded) {
        res.end();
      }
    }

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
    
    if (!res.writableEnded) {
      res.write(`data: ${JSON.stringify(errorEvent)}\n\n`);
      res.end();
    }
  }
}

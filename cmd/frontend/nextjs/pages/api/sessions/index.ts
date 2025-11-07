import { NextApiRequest, NextApiResponse } from 'next';
import { SessionRequest, SessionResponse } from '../../../types';
import { getOrchestratorURL } from '../../../utils/orchestrator';

export default async function handler(
  req: NextApiRequest,
  res: NextApiResponse<SessionResponse | { error: string }>
) {
  if (req.method !== 'POST') {
    return res.status(405).json({ error: 'Method not allowed' });
  }

  try {
    const { topic, explanation_type }: SessionRequest = req.body;

    if (!topic || typeof topic !== 'string' || topic.trim().length === 0) {
      return res.status(400).json({ error: 'Topic is required' });
    }

    // Create session with orchestrator
    const ORCHESTRATOR_URL = getOrchestratorURL();
    console.log(`[Session Creation] Connecting to: ${ORCHESTRATOR_URL}/api/sessions`);
    console.log(`[Session Creation] Environment check - ORCHESTRATOR_URL from env: ${process.env.ORCHESTRATOR_URL || 'NOT SET'}`);
    
    const orchestratorResponse = await fetch(`${ORCHESTRATOR_URL}/api/sessions`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ 
        topic: topic.trim(),
        explanation_type: explanation_type || 'standard'
      }),
    });

    if (!orchestratorResponse.ok) {
      const errorText = await orchestratorResponse.text();
      throw new Error(`Orchestrator error: ${orchestratorResponse.status} ${errorText}`);
    }

    const sessionData = await orchestratorResponse.json();

    res.status(200).json({
      session_id: sessionData.id,
      status: 'created',
    });
  } catch (error) {
    console.error('Session creation error:', error);
    res.status(500).json({
      error: error instanceof Error ? error.message : 'Internal server error',
    });
  }
}

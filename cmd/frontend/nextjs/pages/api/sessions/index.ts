import { NextApiRequest, NextApiResponse } from 'next';
import { SessionRequest, SessionResponse } from '../../../types';

const ORCHESTRATOR_URL = process.env.ORCHESTRATOR_URL || 'http://localhost:8080';

export default async function handler(
  req: NextApiRequest,
  res: NextApiResponse<SessionResponse | { error: string }>
) {
  if (req.method !== 'POST') {
    return res.status(405).json({ error: 'Method not allowed' });
  }

  try {
    const { topic }: SessionRequest = req.body;

    if (!topic || typeof topic !== 'string' || topic.trim().length === 0) {
      return res.status(400).json({ error: 'Topic is required' });
    }

    // Create session with orchestrator
    const orchestratorResponse = await fetch(`${ORCHESTRATOR_URL}/api/sessions`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ topic: topic.trim() }),
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

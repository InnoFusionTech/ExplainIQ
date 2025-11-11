import { NextApiRequest, NextApiResponse } from 'next';

export default function handler(
  req: NextApiRequest,
  res: NextApiResponse<{ url: string } | { error: string }>
) {
  // Get orchestrator URL from environment variables (available at runtime in Cloud Run)
  const orchestratorURL = process.env.ORCHESTRATOR_URL || 
                          process.env.NEXT_PUBLIC_ORCHESTRATOR_URL || 
                          'http://localhost:8080';

  res.status(200).json({ url: orchestratorURL });
}


import { ImageResponse } from '@vercel/og';
import { NextRequest } from 'next/server';

export const config = {
  runtime: 'edge',
};

export default async function handler(req: NextRequest) {
  try {
    const { searchParams } = new URL(req.url);
    const pathParts = req.url.split('/brainprint/');
    const userID = pathParts.length > 1 
      ? pathParts[1].split('/image')[0] 
      : searchParams.get('userID') || 'User';
    
    const name = searchParams.get('name') || userID || 'Learner';
    const recommendedType = searchParams.get('type') || 'Standard';
    const totalSessions = searchParams.get('sessions') || '0';

    // Fetch BrainPrint data if available
    let brainPrintData = null;
    if (userID && userID !== 'User') {
      try {
        // Edge runtime has access to runtime env vars
        const orchestratorURL = process.env.ORCHESTRATOR_URL || process.env.NEXT_PUBLIC_ORCHESTRATOR_URL || 'http://localhost:8080';
        const response = await fetch(`${orchestratorURL}/api/brainprint/${userID}`);
        if (response.ok) {
          brainPrintData = await response.json();
        }
      } catch (err) {
        console.error('Failed to fetch BrainPrint data:', err);
      }
    }

    // Use fetched data or fallback to query params
    const displayName = brainPrintData ? (brainPrintData.userID || name) : name;
    const displayType = brainPrintData?.recommendedType || brainPrintData?.dominantStyle || recommendedType;
    const displaySessions = brainPrintData?.totalSessions || parseInt(totalSessions) || 0;
    const usageBreakdown = brainPrintData?.usage || brainPrintData?.byType || {};

    // Get color for each learning type
    const getTypeColor = (type: string) => {
      switch (type.toLowerCase()) {
        case 'standard':
          return '#3B82F6'; // blue
        case 'visualization':
          return '#8B5CF6'; // purple
        case 'simple':
          return '#10B981'; // green
        case 'analogy':
          return '#F59E0B'; // yellow
        default:
          return '#3B82F6'; // blue
      }
    };

    const getTypeEmoji = (type: string) => {
      switch (type.toLowerCase()) {
        case 'standard':
          return '📚';
        case 'visualization':
          return '🎨';
        case 'simple':
          return '💡';
        case 'analogy':
          return '🔗';
        default:
          return '🧠';
      }
    };

    const typeColor = getTypeColor(displayType);
    const typeEmoji = getTypeEmoji(displayType);

    return new ImageResponse(
      (
        <div
          style={{
            height: '100%',
            width: '100%',
            display: 'flex',
            flexDirection: 'column',
            alignItems: 'center',
            justifyContent: 'center',
            backgroundColor: '#f9fafb',
            backgroundImage: 'linear-gradient(to bottom right, #f0f9ff, #e0e7ff)',
            fontFamily: 'system-ui, -apple-system, sans-serif',
          }}
        >
          {/* Header */}
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              marginBottom: '40px',
            }}
          >
            <div
              style={{
                fontSize: '64px',
                marginRight: '20px',
              }}
            >
              🧠
            </div>
            <div
              style={{
                fontSize: '48px',
                fontWeight: 'bold',
                color: '#1f2937',
              }}
            >
              ExplainIQ BrainPrint
            </div>
          </div>

          {/* Main Content Card */}
          <div
            style={{
              display: 'flex',
              flexDirection: 'column',
              alignItems: 'center',
              justifyContent: 'center',
              backgroundColor: 'white',
              borderRadius: '24px',
              padding: '60px 80px',
              boxShadow: '0 20px 25px -5px rgba(0, 0, 0, 0.1), 0 10px 10px -5px rgba(0, 0, 0, 0.04)',
              border: `4px solid ${typeColor}`,
              maxWidth: '800px',
            }}
          >
            {/* User Name */}
            <div
              style={{
                fontSize: '42px',
                fontWeight: 'bold',
                color: '#111827',
                marginBottom: '20px',
              }}
            >
              {displayName}
            </div>

            {/* Dominant Learning Style */}
            <div
              style={{
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                marginBottom: '30px',
              }}
            >
              <div
                style={{
                  fontSize: '56px',
                  marginRight: '16px',
                }}
              >
                {typeEmoji}
              </div>
              <div
                style={{
                  fontSize: '36px',
                  fontWeight: '600',
                  color: typeColor,
                  textTransform: 'capitalize',
                }}
              >
                {displayType} Learner
              </div>
            </div>

            {/* Stats */}
            <div
              style={{
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                fontSize: '28px',
                color: '#6b7280',
                marginTop: '20px',
                marginBottom: '30px',
              }}
            >
              <span style={{ fontWeight: '600', color: '#374151' }}>{displaySessions}</span>
              <span style={{ margin: '0 8px' }}>learning sessions</span>
            </div>

            {/* Usage Breakdown - Bar Chart */}
            {Object.keys(usageBreakdown).length > 0 && (
              <div
                style={{
                  display: 'flex',
                  flexDirection: 'column',
                  width: '100%',
                  marginTop: '20px',
                  paddingTop: '30px',
                  borderTop: '2px solid #e5e7eb',
                }}
              >
                <div
                  style={{
                    fontSize: '24px',
                    fontWeight: '600',
                    color: '#374151',
                    marginBottom: '20px',
                    textAlign: 'center',
                  }}
                >
                  Usage Breakdown
                </div>
                <div
                  style={{
                    display: 'flex',
                    flexDirection: 'column',
                    gap: '12px',
                    width: '100%',
                  }}
                >
                  {Object.entries(usageBreakdown).map(([type, count]: [string, any]) => {
                    if (!count || count === 0) return null;
                    const total = Object.values(usageBreakdown).reduce((sum: number, val: any) => sum + (val || 0), 0);
                    const percentage = total > 0 ? Math.round((count / total) * 100) : 0;
                    const typeColor = getTypeColor(type);
                    return (
                      <div
                        key={type}
                        style={{
                          display: 'flex',
                          flexDirection: 'column',
                          gap: '6px',
                        }}
                      >
                        <div
                          style={{
                            display: 'flex',
                            justifyContent: 'space-between',
                            fontSize: '20px',
                            color: '#374151',
                          }}
                        >
                          <span style={{ fontWeight: '500' }}>{type}</span>
                          <span style={{ fontWeight: '600', color: typeColor }}>
                            {count} ({percentage}%)
                          </span>
                        </div>
                        <div
                          style={{
                            width: '100%',
                            height: '16px',
                            backgroundColor: '#e5e7eb',
                            borderRadius: '8px',
                            overflow: 'hidden',
                          }}
                        >
                          <div
                            style={{
                              width: `${percentage}%`,
                              height: '100%',
                              backgroundColor: typeColor,
                              borderRadius: '8px',
                            }}
                          />
                        </div>
                      </div>
                    );
                  })}
                </div>
              </div>
            )}
          </div>

          {/* Footer */}
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              marginTop: '40px',
              fontSize: '24px',
              color: '#9ca3af',
              fontWeight: '500',
            }}
          >
            My BrainPrint — Powered by ExplainIQ
          </div>
        </div>
      ),
      {
        width: 1200,
        height: 630,
      }
    );
  } catch (e: any) {
    console.error('Error generating BrainPrint image:', e);
    return new Response(`Failed to generate image: ${e.message}`, { status: 500 });
  }
}

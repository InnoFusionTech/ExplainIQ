import { useState, useEffect } from 'react';

interface BrainPrintData {
  userID: string;
  totalSessions: number;
  byType: {
    Standard?: number;
    Visualization?: number;
    Simple?: number;
    Analogy?: number;
  };
  recommendedType: string;
}

interface BrainPrintCardProps {
  userID?: string;
  className?: string;
}

export default function BrainPrintCard({ userID = 'default', className = '' }: BrainPrintCardProps) {
  const [brainPrint, setBrainPrint] = useState<BrainPrintData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchBrainPrint = async () => {
      if (!userID) {
        setLoading(false);
        return;
      }

      try {
        setLoading(true);
        const orchestratorURL = process.env.NEXT_PUBLIC_ORCHESTRATOR_URL || 'http://localhost:8080';
        const response = await fetch(`${orchestratorURL}/api/brainprint/${userID}`);
        
        if (!response.ok) {
          throw new Error(`Failed to fetch BrainPrint: ${response.statusText}`);
        }

        const data = await response.json();
        setBrainPrint(data);
        setError(null);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to load BrainPrint');
        console.error('Error fetching BrainPrint:', err);
      } finally {
        setLoading(false);
      }
    };

    fetchBrainPrint();
  }, [userID]);

  if (loading) {
    return (
      <div className={`bg-white rounded-lg shadow-md p-6 ${className}`}>
        <div className="animate-pulse">
          <div className="h-4 bg-gray-200 rounded w-1/3 mb-4"></div>
          <div className="h-20 bg-gray-200 rounded"></div>
        </div>
      </div>
    );
  }

  if (error || !brainPrint) {
    return (
      <div className={`bg-white rounded-lg shadow-md p-6 ${className}`}>
        <h3 className="text-lg font-semibold text-gray-800 mb-2">Your BrainPrint</h3>
        <p className="text-gray-500 text-sm">
          {error || 'No learning data available yet. Complete a session to see your learning style!'}
        </p>
      </div>
    );
  }

  // Calculate total for pie chart
  const total = Object.values(brainPrint.byType).reduce((sum, count) => sum + (count || 0), 0);
  
  // Calculate percentages for each type
  const getPercentage = (count: number) => total > 0 ? Math.round((count / total) * 100) : 0;

  // Get color for each type
  const getTypeColor = (type: string) => {
    switch (type) {
      case 'Standard':
        return 'bg-blue-500';
      case 'Visualization':
        return 'bg-purple-500';
      case 'Simple':
        return 'bg-green-500';
      case 'Analogy':
        return 'bg-yellow-500';
      default:
        return 'bg-gray-500';
    }
  };

  // Get suggestion message
  const getSuggestion = () => {
    if (brainPrint.totalSessions < 3) {
      return 'Try different explanation types to discover your learning style!';
    }
    
    const recommendation = brainPrint.recommendedType;
    const alternatives = ['Standard', 'Visualization', 'Simple', 'Analogy'].filter(
      t => t !== recommendation && (brainPrint.byType[t as keyof typeof brainPrint.byType] || 0) < 2
    );

    if (alternatives.length > 0) {
      return `You learn best through ${recommendation}. Try ${alternatives[0]} next!`;
    }
    
    return `You learn best through ${recommendation}. Keep exploring!`;
  };

  return (
    <div className={`bg-white rounded-lg shadow-md p-6 ${className}`}>
      <div className="flex items-center justify-between mb-4">
        <h3 className="text-lg font-semibold text-gray-800">🧠 Your BrainPrint</h3>
        <span className="text-xs text-gray-500">{brainPrint.totalSessions} sessions</span>
      </div>

      {/* Dominant Learning Style */}
      <div className="mb-4">
        <p className="text-sm text-gray-600 mb-1">Dominant Learning Style</p>
        <div className="flex items-center space-x-2">
          <span className={`px-3 py-1 rounded-full text-sm font-medium text-white ${getTypeColor(brainPrint.recommendedType)}`}>
            {brainPrint.recommendedType}
          </span>
        </div>
      </div>

      {/* Usage Breakdown */}
      <div className="mb-4">
        <p className="text-sm text-gray-600 mb-2">Explanation Usage</p>
        <div className="space-y-2">
          {Object.entries(brainPrint.byType).map(([type, count]) => {
            if (!count || count === 0) return null;
            const percentage = getPercentage(count);
            return (
              <div key={type} className="flex items-center space-x-2">
                <div className="flex-1">
                  <div className="flex justify-between text-xs mb-1">
                    <span className="text-gray-700">{type}</span>
                    <span className="text-gray-500">{count} ({percentage}%)</span>
                  </div>
                  <div className="w-full bg-gray-200 rounded-full h-2">
                    <div
                      className={`h-2 rounded-full ${getTypeColor(type)} transition-all duration-300`}
                      style={{ width: `${percentage}%` }}
                    ></div>
                  </div>
                </div>
              </div>
            );
          })}
        </div>
      </div>

      {/* Suggestion Message */}
      <div className="mt-4 pt-4 border-t border-gray-200">
        <p className="text-sm text-gray-600 italic mb-3">
          💡 {getSuggestion()}
        </p>
        
        {/* Share Button */}
        {brainPrint && (
          <button
            onClick={() => {
              const imageURL = `/api/brainprint/${userID}/image?name=${encodeURIComponent(userID)}&type=${encodeURIComponent(brainPrint.recommendedType)}&sessions=${brainPrint.totalSessions}`;
              window.open(imageURL, '_blank');
            }}
            className="w-full mt-3 px-4 py-2 bg-gradient-to-r from-blue-600 to-indigo-600 text-white rounded-lg font-medium hover:from-blue-700 hover:to-indigo-700 transition-all duration-200 flex items-center justify-center space-x-2 shadow-md hover:shadow-lg transform hover:-translate-y-0.5"
          >
            <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
            </svg>
            <span>Share Your BrainPrint</span>
          </button>
        )}
      </div>
    </div>
  );
}


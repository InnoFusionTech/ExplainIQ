import { useState, useEffect, useRef } from 'react';
import { getOrchestratorURL } from '../utils/getOrchestratorURL';

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
  dominantStyle?: string;
  tip?: string;
}

interface BrainPrintCardProps {
  userID?: string;
  className?: string;
  onUpdate?: (data: BrainPrintData) => void;
}

export default function BrainPrintCard({ userID = 'default', className = '', onUpdate }: BrainPrintCardProps) {
  const [brainPrint, setBrainPrint] = useState<BrainPrintData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [isUpdating, setIsUpdating] = useState(false);
  const refreshRef = useRef<(() => void) | null>(null);

  const fetchBrainPrint = async () => {
    if (!userID) {
      setLoading(false);
      return;
    }

    try {
      setLoading(true);
      const orchestratorURL = getOrchestratorURL();
      const response = await fetch(`${orchestratorURL}/api/brainprint/${userID}`);
      
      if (!response.ok) {
        throw new Error(`Failed to fetch BrainPrint: ${response.statusText}`);
      }

      const data = await response.json();
      setBrainPrint(data);
      setError(null);
      if (onUpdate) {
        onUpdate(data);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load BrainPrint');
      console.error('Error fetching BrainPrint:', err);
    } finally {
      setLoading(false);
      setIsUpdating(false);
    }
  };

  useEffect(() => {
    fetchBrainPrint();
    
    // Auto-refresh every 30 seconds to get updated data
    const interval = setInterval(fetchBrainPrint, 30000);
    
    // Expose refresh function
    refreshRef.current = fetchBrainPrint;
    
    // Listen for refresh events
    const handleRefresh = (event: Event) => {
      const customEvent = event as CustomEvent;
      if (customEvent.detail?.userID === userID || !customEvent.detail?.userID) {
        fetchBrainPrint();
      }
    };
    window.addEventListener('brainprint-refresh', handleRefresh);
    
    return () => {
      clearInterval(interval);
      window.removeEventListener('brainprint-refresh', handleRefresh);
    };
  }, [userID]);

  // Expose refresh function to parent via onUpdate callback
  useEffect(() => {
    if (onUpdate && brainPrint) {
      onUpdate(brainPrint);
    }
  }, [brainPrint, onUpdate]);

  // Refresh when component receives update signal
  const handleRefresh = () => {
    setIsUpdating(true);
    fetchBrainPrint();
  };

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
        return { bg: 'bg-blue-500', text: 'text-blue-700', border: 'border-blue-500', gradient: 'from-blue-500 to-blue-600' };
      case 'Visualization':
        return { bg: 'bg-purple-500', text: 'text-purple-700', border: 'border-purple-500', gradient: 'from-purple-500 to-indigo-600' };
      case 'Simple':
        return { bg: 'bg-green-500', text: 'text-green-700', border: 'border-green-500', gradient: 'from-green-500 to-emerald-600' };
      case 'Analogy':
        return { bg: 'bg-yellow-500', text: 'text-yellow-700', border: 'border-yellow-500', gradient: 'from-yellow-500 to-orange-500' };
      default:
        return { bg: 'bg-gray-500', text: 'text-gray-700', border: 'border-gray-500', gradient: 'from-gray-500 to-gray-600' };
    }
  };

  const dominantStyle = brainPrint.dominantStyle || brainPrint.recommendedType;
  const colors = getTypeColor(dominantStyle);

  // Calculate progress percentage (based on total sessions, max 100)
  const progressPercentage = Math.min((brainPrint.totalSessions / 20) * 100, 100);

  return (
    <div className={`bg-white rounded-lg shadow-md p-6 ${className} transition-all duration-300 ${isUpdating ? 'animate-pulse' : ''}`}>
      <div className="flex items-center justify-between mb-4">
        <h3 className="text-lg font-semibold text-gray-800 flex items-center space-x-2">
          <span className="text-2xl animate-pulse">🧠</span>
          <span>Your BrainPrint</span>
        </h3>
        <div className="flex items-center space-x-2">
          <span className="text-xs text-gray-500">{brainPrint.totalSessions} sessions</span>
          <button
            onClick={handleRefresh}
            className="text-gray-400 hover:text-gray-600 transition-colors"
            title="Refresh BrainPrint"
          >
            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
            </svg>
          </button>
        </div>
      </div>

      {/* Dominant Learning Style with Animation */}
      <div className="mb-6">
        <p className="text-sm text-gray-600 mb-2">Dominant Learning Style</p>
        <div className="flex items-center space-x-3">
          <div className={`relative w-16 h-16 rounded-full bg-gradient-to-br ${colors.gradient} flex items-center justify-center shadow-lg transform transition-transform duration-300 hover:scale-110`}>
            <span className="text-2xl text-white font-bold">
              {dominantStyle === 'Visualization' ? '📊' : 
               dominantStyle === 'Simple' ? '📝' : 
               dominantStyle === 'Analogy' ? '💡' : '📚'}
            </span>
            {/* Animated progress ring */}
            <svg className="absolute inset-0 w-16 h-16 transform -rotate-90" viewBox="0 0 64 64">
              <circle
                cx="32"
                cy="32"
                r="28"
                fill="none"
                stroke="rgba(255,255,255,0.3)"
                strokeWidth="4"
              />
              <circle
                cx="32"
                cy="32"
                r="28"
                fill="none"
                stroke="white"
                strokeWidth="4"
                strokeDasharray={`${2 * Math.PI * 28}`}
                strokeDashoffset={`${2 * Math.PI * 28 * (1 - progressPercentage / 100)}`}
                className="transition-all duration-1000 ease-out"
                strokeLinecap="round"
              />
            </svg>
          </div>
          <div className="flex-1">
            <span className={`px-4 py-2 rounded-full text-sm font-bold text-white bg-gradient-to-r ${colors.gradient} shadow-md inline-block animate-fade-in`}>
              {dominantStyle}
            </span>
            <p className="text-xs text-gray-500 mt-1">
              {brainPrint.totalSessions} {brainPrint.totalSessions === 1 ? 'session' : 'sessions'} completed
            </p>
          </div>
        </div>
      </div>

      {/* Animated Progress Bar */}
      <div className="mb-6">
        <div className="flex justify-between text-xs mb-2">
          <span className="text-gray-600 font-medium">Overall Progress</span>
          <span className="text-gray-500">{Math.round(progressPercentage)}%</span>
        </div>
        <div className="w-full bg-gray-200 rounded-full h-3 overflow-hidden">
          <div
            className={`h-3 rounded-full bg-gradient-to-r ${colors.gradient} transition-all duration-1000 ease-out shadow-sm`}
            style={{ width: `${progressPercentage}%` }}
          >
            <div className="h-full w-full bg-gradient-to-r from-transparent via-white/20 to-transparent animate-shimmer"></div>
          </div>
        </div>
      </div>

      {/* Usage Breakdown with Animated Bars */}
      <div className="mb-6">
        <p className="text-sm text-gray-600 mb-3 font-medium">Explanation Usage</p>
        <div className="space-y-3">
          {Object.entries(brainPrint.byType).map(([type, count]) => {
            if (!count || count === 0) return null;
            const percentage = getPercentage(count);
            const typeColors = getTypeColor(type);
            return (
              <div key={type} className="group">
                <div className="flex justify-between text-xs mb-1">
                  <span className="text-gray-700 font-medium">{type}</span>
                  <span className="text-gray-500">{count} ({percentage}%)</span>
                </div>
                <div className="w-full bg-gray-200 rounded-full h-2.5 overflow-hidden">
                  <div
                    className={`h-2.5 rounded-full bg-gradient-to-r ${typeColors.gradient} transition-all duration-500 ease-out shadow-sm group-hover:shadow-md`}
                    style={{ width: `${percentage}%` }}
                  >
                    <div className="h-full w-full bg-gradient-to-r from-transparent via-white/20 to-transparent animate-shimmer"></div>
                  </div>
                </div>
              </div>
            );
          })}
        </div>
      </div>

      {/* Tip of the Day */}
      {brainPrint.tip && (
        <div className="mb-6 p-4 bg-gradient-to-r from-blue-50 to-indigo-50 rounded-lg border border-blue-200 animate-fade-in">
          <div className="flex items-start space-x-2">
            <span className="text-xl">💡</span>
            <div className="flex-1">
              <p className="text-xs font-semibold text-blue-900 mb-1">Tip of the Day</p>
              <p className="text-sm text-blue-800 italic">{brainPrint.tip}</p>
            </div>
          </div>
        </div>
      )}

      {/* Share Button */}
      <div className="pt-4 border-t border-gray-200">
        <button
          onClick={() => {
            const imageURL = `/api/brainprint/${userID}/image?name=${encodeURIComponent(userID)}&type=${encodeURIComponent(dominantStyle)}&sessions=${brainPrint.totalSessions}`;
            window.open(imageURL, '_blank');
          }}
          className="w-full px-4 py-3 bg-gradient-to-r from-blue-600 to-indigo-600 text-white rounded-lg font-medium hover:from-blue-700 hover:to-indigo-700 transition-all duration-200 flex items-center justify-center space-x-2 shadow-md hover:shadow-lg transform hover:-translate-y-0.5 active:scale-95"
        >
          <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
          </svg>
          <span>Share Your BrainPrint</span>
        </button>
      </div>
    </div>
  );
}

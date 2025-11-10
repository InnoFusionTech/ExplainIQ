import React, { useState, useEffect } from 'react';
import { ExplanationType } from '../types';
import Link from 'next/link';
import { getOrchestratorURL } from '../utils/getOrchestratorURL';

interface SidebarProps {
  activeType: ExplanationType;
  onTypeChange: (type: ExplanationType) => void;
  disabled?: boolean;
  userID?: string;
}

interface MenuItem {
  id: ExplanationType;
  name: string;
  icon: string;
  description: string;
  color: string;
}

const menuItems: MenuItem[] = [
  {
    id: 'standard',
    name: 'Standard Explanation',
    icon: '📚',
    description: 'Comprehensive structured lesson',
    color: 'text-blue-600 hover:bg-blue-50 border-blue-500',
  },
  {
    id: 'visualization',
    name: 'Visualization',
    icon: '📊',
    description: 'Charts and visual diagrams',
    color: 'text-purple-600 hover:bg-purple-50 border-purple-500',
  },
  {
    id: 'simple',
    name: 'Simple Explanation',
    icon: '💡',
    description: 'Easy-to-understand explanations',
    color: 'text-green-600 hover:bg-green-50 border-green-500',
  },
  {
    id: 'analogy',
    name: 'Analogies',
    icon: '🔗',
    description: 'Real-world comparisons',
    color: 'text-orange-600 hover:bg-orange-50 border-orange-500',
  },
];

interface SavedLesson {
  id: string;
  title: string;
  topic: string;
  explanation_type: string;
  created_at: string;
}

const Sidebar: React.FC<SidebarProps> = ({ activeType, onTypeChange, disabled = false, userID = 'default' }) => {
  const [savedLessons, setSavedLessons] = useState<SavedLesson[]>([]);
  const [isLoadingSaved, setIsLoadingSaved] = useState(false);
  const [showSaved, setShowSaved] = useState(false);

  const fetchSavedLessons = async () => {
    if (!userID) return;
    
    setIsLoadingSaved(true);
    try {
      const orchestratorURL = getOrchestratorURL();
      const response = await fetch(`${orchestratorURL}/api/saved/${userID}`);
      if (response.ok) {
        const data = await response.json();
        setSavedLessons(data.lessons || []);
      }
    } catch (error) {
      console.error('Failed to fetch saved lessons:', error);
    } finally {
      setIsLoadingSaved(false);
    }
  };

  useEffect(() => {
    fetchSavedLessons();
    
    // Listen for refresh events
    const handleRefresh = () => {
      fetchSavedLessons();
    };
    window.addEventListener('saved-lessons-refresh', handleRefresh);
    
    return () => {
      window.removeEventListener('saved-lessons-refresh', handleRefresh);
    };
  }, [userID]);

  const getTypeIcon = (type: string) => {
    switch (type?.toLowerCase()) {
      case 'standard': return '📚';
      case 'visualization': return '📊';
      case 'simple': return '💡';
      case 'analogy': return '🔗';
      default: return '📝';
    }
  };

  return (
    <div className="fixed left-0 top-0 h-full w-64 bg-white shadow-xl border-r border-gray-200 z-40 overflow-y-auto animate-slide-in">
      <div className="p-6">
        <div className="mb-6">
          <h2 className="text-xl font-bold text-gray-900 mb-1">Explanation Types</h2>
          <p className="text-xs text-gray-500">Choose your preferred learning style</p>
        </div>
        <nav className="space-y-2">
          {menuItems.map((item) => {
            const isActive = activeType === item.id;
            return (
              <button
                key={item.id}
                onClick={() => !disabled && onTypeChange(item.id)}
                disabled={disabled}
                className={`
                  w-full text-left px-4 py-3 rounded-lg transition-all duration-300
                  ${isActive 
                    ? `${item.color.split(' ')[0]} bg-opacity-10 border-l-4 font-semibold shadow-sm transform scale-[1.02]` 
                    : 'text-gray-700 hover:bg-gray-50 border-l-4 border-transparent hover:shadow-sm'
                  }
                  ${disabled ? 'opacity-50 cursor-not-allowed' : 'cursor-pointer'}
                  focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2
                `}
              >
                <div className="flex items-start space-x-3">
                  <span className="text-2xl">{item.icon}</span>
                  <div className="flex-1 min-w-0">
                    <div className={`text-sm font-medium ${isActive ? item.color.split(' ')[0] : 'text-gray-900'}`}>
                      {item.name}
                    </div>
                    <div className="text-xs text-gray-500 mt-1">{item.description}</div>
                  </div>
                  {isActive && (
                    <svg className="w-5 h-5 flex-shrink-0" fill="currentColor" viewBox="0 0 20 20">
                      <path fillRule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clipRule="evenodd" />
                    </svg>
                  )}
                </div>
              </button>
            );
          })}
        </nav>

        {/* Saved Lessons Section */}
        <div className="mt-8 border-t border-gray-200 pt-6">
          <button
            onClick={() => setShowSaved(!showSaved)}
            className="w-full flex items-center justify-between text-left px-4 py-3 rounded-lg transition-all duration-200 hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2"
          >
            <div className="flex items-center space-x-3">
              <span className="text-2xl">💾</span>
              <div>
                <div className="text-sm font-semibold text-gray-900">Saved Lessons</div>
                <div className="text-xs text-gray-500">{savedLessons.length} saved</div>
              </div>
            </div>
            <svg
              className={`w-5 h-5 text-gray-500 transition-transform duration-200 ${showSaved ? 'transform rotate-180' : ''}`}
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
            </svg>
          </button>

          {showSaved && (
            <div className="mt-3 space-y-2 animate-fade-in">
              {isLoadingSaved ? (
                <div className="text-center py-4">
                  <svg className="animate-spin h-5 w-5 text-gray-400 mx-auto" fill="none" viewBox="0 0 24 24">
                    <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
                    <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                  </svg>
                </div>
              ) : savedLessons.length === 0 ? (
                <div className="text-center py-4 text-sm text-gray-500">
                  No saved lessons yet
                </div>
              ) : (
                savedLessons.map((lesson) => (
                  <Link
                    key={lesson.id}
                    href={`/saved/${userID}/${lesson.id}`}
                    className="block px-4 py-3 rounded-lg transition-all duration-200 hover:bg-gray-50 border border-transparent hover:border-gray-200 group"
                  >
                    <div className="flex items-start space-x-3">
                      <span className="text-xl flex-shrink-0">{getTypeIcon(lesson.explanation_type)}</span>
                      <div className="flex-1 min-w-0">
                        <div className="text-sm font-medium text-gray-900 truncate group-hover:text-blue-600">
                          {lesson.title || lesson.topic}
                        </div>
                        <div className="text-xs text-gray-500 mt-1">
                          {new Date(lesson.created_at).toLocaleDateString()}
                        </div>
                      </div>
                    </div>
                  </Link>
                ))
              )}
            </div>
          )}
        </div>

        {/* Info Section */}
        <div className="mt-8 p-4 bg-blue-50 rounded-lg border border-blue-200">
          <h3 className="text-sm font-medium text-blue-900 mb-2">💡 Tip</h3>
          <p className="text-xs text-blue-800">
            Choose an explanation type that matches your learning style. Each type provides a different approach to understanding the topic.
          </p>
        </div>
      </div>
    </div>
  );
};

export default Sidebar;


import React from 'react';
import { ExplanationType } from '../types';

interface SidebarProps {
  activeType: ExplanationType;
  onTypeChange: (type: ExplanationType) => void;
  disabled?: boolean;
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

const Sidebar: React.FC<SidebarProps> = ({ activeType, onTypeChange, disabled = false }) => {
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


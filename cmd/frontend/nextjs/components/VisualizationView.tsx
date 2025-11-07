import React, { useState, useEffect } from 'react';
import { OGLesson } from '../types';

interface VisualizationViewProps {
  lesson: OGLesson;
  topic: string;
}

interface ChartData {
  labels: string[];
  data: number[];
}

const VisualizationView: React.FC<VisualizationViewProps> = ({ lesson, topic }) => {
  const [chartData, setChartData] = useState<ChartData | null>(null);

  useEffect(() => {
    // Generate chart data from lesson sections
    const sections = [
      { name: 'Big Picture', length: lesson.big_picture?.length || 0 },
      { name: 'Metaphor', length: lesson.metaphor?.length || 0 },
      { name: 'Core Mechanism', length: lesson.core_mechanism?.length || 0 },
      { name: 'Memory Hook', length: lesson.memory_hook?.length || 0 },
      { name: 'Real Life', length: lesson.real_life?.length || 0 },
      { name: 'Best Practices', length: lesson.best_practices?.length || 0 },
    ];

    setChartData({
      labels: sections.map(s => s.name),
      data: sections.map(s => s.length),
    });
  }, [lesson]);

  return (
    <div className="space-y-6 animate-fade-in">
      {/* Topic Header */}
      <div className="bg-gradient-to-r from-purple-500 via-indigo-600 to-purple-600 text-white p-6 rounded-lg shadow-lg transform hover:scale-[1.01] transition-transform duration-200">
        <h2 className="text-2xl font-bold mb-2">{topic}</h2>
        <p className="text-purple-100 font-medium">Visual Representation</p>
      </div>

      {/* Bar Chart */}
      {chartData && (
        <div className="bg-white rounded-lg shadow-md p-6">
          <h3 className="text-lg font-semibold text-gray-900 mb-4">Section Length Distribution</h3>
          <div className="space-y-3">
            {chartData.labels.map((label, index) => {
              const maxValue = Math.max(...chartData.data);
              const percentage = (chartData.data[index] / maxValue) * 100;
              return (
                <div key={label}>
                  <div className="flex items-center justify-between mb-1">
                    <span className="text-sm font-medium text-gray-700">{label}</span>
                    <span className="text-sm text-gray-500">{chartData.data[index]} chars</span>
                  </div>
                  <div className="w-full bg-gray-200 rounded-full h-6 overflow-hidden">
                    <div
                      className="bg-gradient-to-r from-purple-500 to-indigo-500 h-6 rounded-full transition-all duration-500 flex items-center justify-end pr-2"
                      style={{ width: `${percentage}%` }}
                    >
                      <span className="text-xs text-white font-medium">{Math.round(percentage)}%</span>
                    </div>
                  </div>
                </div>
              );
            })}
          </div>
        </div>
      )}

      {/* Concept Flow Diagram */}
      <div className="bg-white rounded-lg shadow-md p-6">
        <h3 className="text-lg font-semibold text-gray-900 mb-4">Learning Flow</h3>
        <div className="flex flex-wrap items-center justify-center gap-4">
          <div className="bg-blue-100 px-4 py-2 rounded-lg text-sm font-medium text-blue-900">
            Big Picture
          </div>
          <span className="text-gray-400">→</span>
          <div className="bg-green-100 px-4 py-2 rounded-lg text-sm font-medium text-green-900">
            Metaphor
          </div>
          <span className="text-gray-400">→</span>
          <div className="bg-purple-100 px-4 py-2 rounded-lg text-sm font-medium text-purple-900">
            Core Mechanism
          </div>
          <span className="text-gray-400">→</span>
          <div className="bg-orange-100 px-4 py-2 rounded-lg text-sm font-medium text-orange-900">
            Examples
          </div>
          <span className="text-gray-400">→</span>
          <div className="bg-indigo-100 px-4 py-2 rounded-lg text-sm font-medium text-indigo-900">
            Applications
          </div>
        </div>
      </div>

      {/* Visual Summary */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <div className="bg-white rounded-lg shadow-md p-6 border-l-4 border-purple-500">
          <h3 className="text-lg font-semibold text-gray-900 mb-2">Key Concepts</h3>
          <div className="space-y-2">
            {lesson.big_picture && (
              <div className="text-sm text-gray-700">
                {lesson.big_picture.substring(0, 150)}...
              </div>
            )}
          </div>
        </div>
        <div className="bg-white rounded-lg shadow-md p-6 border-l-4 border-indigo-500">
          <h3 className="text-lg font-semibold text-gray-900 mb-2">Core Principle</h3>
          <div className="text-sm text-gray-700">
            {lesson.core_mechanism?.substring(0, 150)}...
          </div>
        </div>
      </div>
    </div>
  );
};

export default VisualizationView;


import React, { useState, useEffect } from 'react';
import { OGLesson, ImageRef } from '../types';
import InteractiveVisualizations from './InteractiveVisualizations';

interface VisualizationViewProps {
  lesson: OGLesson;
  topic: string;
  images?: ImageRef[];
}

interface ChartData {
  labels: string[];
  data: number[];
}

const VisualizationView: React.FC<VisualizationViewProps> = ({ lesson, topic, images = [] }) => {
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
      {/* Interactive Visualizations */}
      <InteractiveVisualizations lesson={lesson} topic={topic} />
      
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

      {/* Images Section */}
      {images && images.length > 0 && (
        <div className="bg-gradient-to-br from-purple-50 to-indigo-50 rounded-lg p-6 border border-purple-200">
          <h3 className="text-xl font-bold text-gray-900 mb-6 flex items-center space-x-2">
            <span className="text-2xl">🎨</span>
            <span>Visualizations</span>
            <span className="text-sm font-normal text-gray-600 ml-2">({images.length} {images.length === 1 ? 'diagram' : 'diagrams'})</span>
          </h3>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
            {images.map((image, index) => (
              <div 
                key={index} 
                className="bg-white rounded-lg shadow-md overflow-hidden transform hover:scale-[1.02] transition-all duration-300 hover:shadow-xl group"
              >
                <div className="relative overflow-hidden bg-gray-100">
                  <img
                    src={image.url}
                    alt={image.alt_text || image.caption || `Visualization ${index + 1}`}
                    className="w-full h-48 object-cover transition-transform duration-300 group-hover:scale-110"
                    onError={(e) => {
                      const target = e.target as HTMLImageElement;
                      target.style.display = 'none';
                      const parent = target.parentElement;
                      if (parent) {
                        parent.innerHTML = `
                          <div class="w-full h-48 flex items-center justify-center bg-gray-200">
                            <p class="text-gray-500 text-sm">Image failed to load</p>
                          </div>
                        `;
                      }
                    }}
                  />
                  <div className="absolute inset-0 bg-gradient-to-t from-black/60 via-black/20 to-transparent opacity-0 group-hover:opacity-100 transition-opacity duration-300"></div>
                  <div className="absolute top-2 right-2 opacity-0 group-hover:opacity-100 transition-opacity duration-300">
                    <button
                      onClick={() => {
                        const link = document.createElement('a');
                        link.href = image.url;
                        link.download = `visualization-${index + 1}.png`;
                        link.target = '_blank';
                        document.body.appendChild(link);
                        link.click();
                        document.body.removeChild(link);
                      }}
                      className="bg-white/90 hover:bg-white text-gray-800 p-2 rounded-full shadow-lg transition-all duration-200 transform hover:scale-110"
                      title="Download diagram"
                    >
                      <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
                      </svg>
                    </button>
                  </div>
                </div>
                <div className="p-4">
                  <p className="text-sm text-gray-700 font-medium line-clamp-2">{image.caption || image.alt_text || `Visualization ${index + 1}`}</p>
                  {image.alt_text && image.alt_text !== image.caption && (
                    <p className="text-xs text-gray-500 mt-1 line-clamp-1">{image.alt_text}</p>
                  )}
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
};

export default VisualizationView;


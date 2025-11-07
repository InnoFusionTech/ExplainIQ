import React from 'react';
import { OGLesson } from '../types';

interface SimpleExplanationViewProps {
  lesson: OGLesson;
  topic: string;
}

const SimpleExplanationView: React.FC<SimpleExplanationViewProps> = ({ lesson, topic }) => {
  return (
    <div className="space-y-6 animate-fade-in">
      {/* Topic Header */}
      <div className="bg-gradient-to-r from-green-500 via-emerald-500 to-green-600 text-white p-6 rounded-lg shadow-lg transform hover:scale-[1.01] transition-transform duration-200">
        <h2 className="text-2xl font-bold mb-2">{topic}</h2>
        <p className="text-green-100 font-medium">Simple & Easy Explanation</p>
      </div>

      {/* Simple Explanation */}
      {lesson.big_picture && (
        <div className="bg-white rounded-lg shadow-md p-6 border-l-4 border-green-500">
          <div className="flex items-start space-x-3 mb-4">
            <span className="text-3xl">💡</span>
            <div className="flex-1">
              <h3 className="text-xl font-semibold text-gray-900 mb-2">What is it?</h3>
              <p className="text-gray-700 leading-relaxed whitespace-pre-wrap">
                {lesson.big_picture}
              </p>
            </div>
          </div>
        </div>
      )}

      {/* Analogy Section */}
      {lesson.metaphor && (
        <div className="bg-white rounded-lg shadow-md p-6 border-l-4 border-orange-500">
          <div className="flex items-start space-x-3 mb-4">
            <span className="text-3xl">🔗</span>
            <div className="flex-1">
              <h3 className="text-xl font-semibold text-gray-900 mb-2">Think of it like...</h3>
              <p className="text-gray-700 leading-relaxed whitespace-pre-wrap">
                {lesson.metaphor}
              </p>
            </div>
          </div>
        </div>
      )}

      {/* How it Works - Simplified */}
      {lesson.core_mechanism && (
        <div className="bg-white rounded-lg shadow-md p-6 border-l-4 border-blue-500">
          <div className="flex items-start space-x-3 mb-4">
            <span className="text-3xl">⚙️</span>
            <div className="flex-1">
              <h3 className="text-xl font-semibold text-gray-900 mb-2">How it works</h3>
              <p className="text-gray-700 leading-relaxed whitespace-pre-wrap">
                {lesson.core_mechanism}
              </p>
            </div>
          </div>
        </div>
      )}

      {/* Simple Example */}
      {lesson.toy_example_code && (
        <div className="bg-white rounded-lg shadow-md p-6 border-l-4 border-purple-500">
          <div className="flex items-start space-x-3 mb-4">
            <span className="text-3xl">📝</span>
            <div className="flex-1">
              <h3 className="text-xl font-semibold text-gray-900 mb-2">Simple Example</h3>
              <pre className="bg-gray-50 p-4 rounded-lg text-sm overflow-x-auto border border-gray-200">
                <code className="text-gray-800">{lesson.toy_example_code}</code>
              </pre>
            </div>
          </div>
        </div>
      )}

      {/* Real-World Use */}
      {lesson.real_life && (
        <div className="bg-white rounded-lg shadow-md p-6 border-l-4 border-indigo-500">
          <div className="flex items-start space-x-3 mb-4">
            <span className="text-3xl">🌍</span>
            <div className="flex-1">
              <h3 className="text-xl font-semibold text-gray-900 mb-2">Where you see it</h3>
              <p className="text-gray-700 leading-relaxed whitespace-pre-wrap">
                {lesson.real_life}
              </p>
            </div>
          </div>
        </div>
      )}

      {/* Quick Tips */}
      {lesson.best_practices && (
        <div className="bg-gradient-to-r from-yellow-50 to-orange-50 rounded-lg shadow-md p-6 border-l-4 border-yellow-500">
          <div className="flex items-start space-x-3 mb-4">
            <span className="text-3xl">⭐</span>
            <div className="flex-1">
              <h3 className="text-xl font-semibold text-gray-900 mb-2">Quick Tips</h3>
              <p className="text-gray-700 leading-relaxed whitespace-pre-wrap">
                {lesson.best_practices}
              </p>
            </div>
          </div>
        </div>
      )}

      {/* Memory Aid */}
      {lesson.memory_hook && (
        <div className="bg-gradient-to-r from-pink-50 to-rose-50 rounded-lg shadow-md p-6 border-l-4 border-pink-500">
          <div className="flex items-start space-x-3 mb-4">
            <span className="text-3xl">🎯</span>
            <div className="flex-1">
              <h3 className="text-xl font-semibold text-gray-900 mb-2">Remember this</h3>
              <p className="text-gray-700 leading-relaxed whitespace-pre-wrap font-medium">
                {lesson.memory_hook}
              </p>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default SimpleExplanationView;


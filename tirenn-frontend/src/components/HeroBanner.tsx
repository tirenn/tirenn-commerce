import React from 'react';

export const HeroBanner: React.FC = () => {
  return (
    <div className="mb-6 bg-slate-900 text-white rounded-xl p-6 sm:p-8 flex flex-col sm:flex-row sm:items-center justify-between gap-4">
      <div>
        <h1 className="text-2xl sm:text-3xl font-bold tracking-tight text-white mb-1">
          Welcome to Tirenn Commerce
        </h1>
        <p className="text-slate-400 text-xs sm:text-sm max-w-lg">
          Browse products across electronics, fashion, home, and sports with live inventory and fast checkout.
        </p>
      </div>
    </div>
  );
};

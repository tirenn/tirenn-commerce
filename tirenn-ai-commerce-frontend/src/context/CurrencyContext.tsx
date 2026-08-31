import React, { createContext, useContext, useState, useEffect } from 'react';
import { useTranslation } from 'react-i18next';

export type Currency = 'IDR' | 'USD';

interface CurrencyContextType {
  currency: Currency;
  setCurrency: (c: Currency) => void;
  exchangeRate: number; // 1 USD = X IDR
  formatPrice: (price: number, baseCurrency?: string) => string;
  getPriceInActiveCurrency: (price: number, baseCurrency?: string) => number;
  convertToUSD: (amountInIDR: number) => number;
  convertToIDR: (amountInUSD: number) => number;
  isFetchingRate: boolean;
}

const CurrencyContext = createContext<CurrencyContextType | undefined>(undefined);

const DEFAULT_USD_RATE = Number(import.meta.env.VITE_DEFAULT_EXCHANGE_RATE) || 16000.0;
const EXCHANGE_RATE_API_URL = import.meta.env.VITE_EXCHANGE_RATE_API_URL || 'https://open.er-api.com/v6/latest/USD';
const CACHE_MINUTES = Number(import.meta.env.VITE_EXCHANGE_RATE_CACHE_MINUTES) || 60;
const RATE_STORAGE_KEY = 'tirenn_usd_idr_rate';
const RATE_TIMESTAMP_KEY = 'tirenn_usd_idr_rate_time';

export const CurrencyProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const { i18n } = useTranslation();
  
  // Initial currency based on active language (en -> USD, id -> IDR)
  const [currency, setCurrencyState] = useState<Currency>(() => {
    return i18n.language === 'en' ? 'USD' : 'IDR';
  });

  const [exchangeRate, setExchangeRate] = useState<number>(() => {
    const saved = localStorage.getItem(RATE_STORAGE_KEY);
    return saved ? parseFloat(saved) : DEFAULT_USD_RATE;
  });

  const [isFetchingRate, setIsFetchingRate] = useState(false);

  // Fetch live exchange rates from configured free open API
  useEffect(() => {
    const fetchRate = async () => {
      const savedTime = localStorage.getItem(RATE_TIMESTAMP_KEY);
      const now = Date.now();
      
      // Cache rate for configured duration (default: 60 minutes)
      if (savedTime && now - parseInt(savedTime, 10) < CACHE_MINUTES * 60 * 1000) {
        return;
      }

      try {
        setIsFetchingRate(true);
        const res = await fetch(EXCHANGE_RATE_API_URL);
        if (res.ok) {
          const data = await res.json();
          if (data && data.rates && data.rates.IDR) {
            const rate = parseFloat(data.rates.IDR);
            if (rate > 0) {
              setExchangeRate(rate);
              localStorage.setItem(RATE_STORAGE_KEY, rate.toString());
              localStorage.setItem(RATE_TIMESTAMP_KEY, now.toString());
            }
          }
        }
      } catch (err) {
        console.warn('Could not fetch live exchange rate, using fallback rate:', DEFAULT_USD_RATE, err);
      } finally {
        setIsFetchingRate(false);
      }
    };

    fetchRate();
  }, []);

  // Synchronize currency whenever i18n language changes
  useEffect(() => {
    if (i18n.language === 'en') {
      setCurrencyState('USD');
    } else {
      setCurrencyState('IDR');
    }
  }, [i18n.language]);

  const setCurrency = (c: Currency) => {
    setCurrencyState(c);
  };

  const convertToUSD = (amountInIDR: number) => {
    return amountInIDR / (exchangeRate || DEFAULT_USD_RATE);
  };

  const convertToIDR = (amountInUSD: number) => {
    return amountInUSD * (exchangeRate || DEFAULT_USD_RATE);
  };

  const getPriceInActiveCurrency = (price: number, baseCurrency: string = 'IDR'): number => {
    const normBase = (baseCurrency || 'IDR').toUpperCase();
    const rate = exchangeRate || DEFAULT_USD_RATE;

    if (currency === 'USD') {
      // Target is USD
      if (normBase === 'USD') {
        return price; // Already in USD, no conversion
      }
      return price / rate; // Convert IDR to USD
    } else {
      // Target is IDR
      if (normBase === 'IDR') {
        return price; // Already in IDR, no conversion
      }
      return price * rate; // Convert USD to IDR
    }
  };

  const formatPrice = (price: number, baseCurrency: string = 'IDR') => {
    const activeValue = getPriceInActiveCurrency(price, baseCurrency);

    if (currency === 'USD') {
      return new Intl.NumberFormat('en-US', {
        style: 'currency',
        currency: 'USD',
        minimumFractionDigits: 2,
        maximumFractionDigits: 2
      }).format(activeValue);
    }

    // Indonesian Rupiah format
    return new Intl.NumberFormat('id-ID', {
      style: 'currency',
      currency: 'IDR',
      maximumFractionDigits: 0
    }).format(activeValue);
  };

  return (
    <CurrencyContext.Provider
      value={{
        currency,
        setCurrency,
        exchangeRate,
        formatPrice,
        getPriceInActiveCurrency,
        convertToUSD,
        convertToIDR,
        isFetchingRate
      }}
    >
      {children}
    </CurrencyContext.Provider>
  );
};

export const useCurrency = () => {
  const context = useContext(CurrencyContext);
  if (!context) {
    throw new Error('useCurrency must be used within a CurrencyProvider');
  }
  return context;
};

import React, { createContext, useContext, useState, useEffect } from "react";
import { api } from "./index";
import { State } from "./types";

interface AppContextType {
  state: State | null;
  refreshState: () => Promise<void>;
  loading: boolean;
}

const AppContext = createContext<AppContextType | null>(null);

export const AppProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [state, setState] = useState<State | null>(null);
  const [loading, setLoading] = useState(true);

  const refreshState = async () => {
    try {
      const res = await api.get<State>("/state");
      setState(res.data);
    } catch (e) {
      console.error(e);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    refreshState();
  }, []);

  return (
    <AppContext.Provider value={{ state, refreshState, loading }}>
      {children}
    </AppContext.Provider>
  );
};

export const useAppContext = () => {
  const ctx = useContext(AppContext);
  if (!ctx) throw new Error("useAppContext must be used within AppProvider");
  return ctx;
};
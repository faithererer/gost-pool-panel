import React, { createContext, useContext, useState, useEffect } from "react";
import { api } from "./index";
import { State } from "./types";

export type NoticeType = "success" | "error" | "info";

export interface Notice {
  id: string;
  type: NoticeType;
  title: string;
  message?: string;
}

interface AppContextType {
  state: State | null;
  refreshState: () => Promise<void>;
  loading: boolean;
  notices: Notice[];
  notify: (notice: Omit<Notice, "id">) => void;
  dismissNotice: (id: string) => void;
}

const AppContext = createContext<AppContextType | null>(null);

export const AppProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [state, setState] = useState<State | null>(null);
  const [loading, setLoading] = useState(true);
  const [notices, setNotices] = useState<Notice[]>([]);

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

  const dismissNotice = (id: string) => {
    setNotices((current) => current.filter((notice) => notice.id !== id));
  };

  const notify = (notice: Omit<Notice, "id">) => {
    const id = `${Date.now()}-${Math.random().toString(16).slice(2)}`;
    setNotices((current) => [...current, { id, ...notice }].slice(-4));
    window.setTimeout(() => dismissNotice(id), notice.type === "error" ? 6500 : 4200);
  };

  useEffect(() => {
    refreshState();
  }, []);

  return (
    <AppContext.Provider value={{ state, refreshState, loading, notices, notify, dismissNotice }}>
      {children}
    </AppContext.Provider>
  );
};

export const useAppContext = () => {
  const ctx = useContext(AppContext);
  if (!ctx) throw new Error("useAppContext must be used within AppProvider");
  return ctx;
};

import { AnimatePresence, motion } from "framer-motion";
import { CheckCircle2, Info, X, XCircle } from "lucide-react";
import { Notice } from "../../api/AppContext";
import { Button } from "./button";

interface ToastViewportProps {
  notices: Notice[];
  onDismiss: (id: string) => void;
}

const variants = {
  success: {
    icon: CheckCircle2,
    className: "border-green-500/30 bg-green-500/10 text-green-100",
    iconClassName: "text-green-400",
  },
  error: {
    icon: XCircle,
    className: "border-destructive/40 bg-destructive/15 text-red-50",
    iconClassName: "text-destructive",
  },
  info: {
    icon: Info,
    className: "border-primary/20 bg-primary/10 text-foreground",
    iconClassName: "text-primary",
  },
};

export function ToastViewport({ notices, onDismiss }: ToastViewportProps) {
  return (
    <div className="pointer-events-none fixed right-4 top-4 z-[80] flex w-[min(420px,calc(100vw-32px))] flex-col gap-3">
      <AnimatePresence>
        {notices.map((notice) => {
          const variant = variants[notice.type];
          const Icon = variant.icon;
          return (
            <motion.div
              key={notice.id}
              initial={{ opacity: 0, x: 24, scale: 0.98 }}
              animate={{ opacity: 1, x: 0, scale: 1 }}
              exit={{ opacity: 0, x: 24, scale: 0.98 }}
              transition={{ duration: 0.18 }}
              className={`pointer-events-auto rounded-lg border p-4 shadow-lg backdrop-blur ${variant.className}`}
            >
              <div className="flex items-start gap-3">
                <Icon className={`mt-0.5 h-5 w-5 shrink-0 ${variant.iconClassName}`} />
                <div className="min-w-0 flex-1">
                  <div className="text-sm font-semibold">{notice.title}</div>
                  {notice.message && <div className="mt-1 text-sm text-muted-foreground">{notice.message}</div>}
                </div>
                <Button variant="ghost" size="icon" className="h-7 w-7 shrink-0" onClick={() => onDismiss(notice.id)}>
                  <X className="h-4 w-4" />
                </Button>
              </div>
            </motion.div>
          );
        })}
      </AnimatePresence>
    </div>
  );
}

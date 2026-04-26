import * as React from "react"

import { motion, AnimatePresence } from "framer-motion"

interface ModalProps {
  isOpen: boolean
  onClose: () => void
  title: string
  description?: string
  children: React.ReactNode
  footer?: React.ReactNode
}

export function Modal({ isOpen, onClose, title, description, children, footer }: ModalProps) {
  return (
    <AnimatePresence>
      {isOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center">
          <motion.div
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            onClick={onClose}
            className="fixed inset-0 bg-background/80 backdrop-blur-sm"
          />
          <motion.div
            initial={{ opacity: 0, scale: 0.95 }}
            animate={{ opacity: 1, scale: 1 }}
            exit={{ opacity: 0, scale: 0.95 }}
            transition={{ duration: 0.2 }}
            className="z-50 w-full max-w-lg rounded-xl border bg-card p-6 shadow-lg sm:max-w-[425px]"
          >
            <div className="flex flex-col space-y-1.5 text-center sm:text-left mb-4">
              <h2 className="text-lg font-semibold leading-none tracking-tight">{title}</h2>
              {description && <p className="text-sm text-muted-foreground">{description}</p>}
            </div>
            <div className="mb-4">
              {children}
            </div>
            {footer && (
              <div className="flex flex-col-reverse sm:flex-row sm:justify-end sm:space-x-2">
                {footer}
              </div>
            )}
          </motion.div>
        </div>
      )}
    </AnimatePresence>
  )
}
import { useEffect, useState, createContext, useContext, type ReactNode } from "react"
import { getLocale, getTextDirection, setLocale, type Locale } from "@/paraglide/runtime"
import { DirectionProvider } from "@/components/ui/direction"

interface LanguageContextType {
  locale: Locale
  dir: "ltr" | "rtl"
  changeLanguage: (newLocale: Locale) => void
}

const LanguageContext = createContext<LanguageContextType | undefined>(undefined)

export function LanguageProvider({ children }: { children: ReactNode }) {
  const [locale, setLocalState] = useState<Locale>(getLocale())

  const dir = getTextDirection(locale)

  const changeLanguage = (newLocale: Locale) => {
    setLocale(newLocale, { reload: false })
    setLocalState(newLocale)
  }

  useEffect(() => {
    document.documentElement.lang = locale
    document.documentElement.dir = dir
  }, [locale, dir])

  return (
    <LanguageContext.Provider value={{ locale, dir, changeLanguage }}>
      <DirectionProvider dir={dir}>
        <div key={locale}>
          {children}
        </div>
      </DirectionProvider>
    </LanguageContext.Provider>
  )
}

export const useLanguage = () => {
  const context = useContext(LanguageContext)
  if (!context) throw new Error("useLanguage must be used within LanguageProvider")
  return context
}

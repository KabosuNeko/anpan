import React, {createContext, type ReactNode, useContext} from 'react'

/**
 * Minimal terminal-native theme.
 * Does not force background or foreground colors, preserving full terminal transparency.
 */
export type Palette = {
  primary?: string
  muted?: string
  dimAccent: boolean
}

export const defaultPalette: Palette = {
  primary: undefined,
  muted: undefined,
  dimAccent: true,
}

const PaletteContext = createContext<Palette>(defaultPalette)

export function ThemeProvider({children}: {children: ReactNode}) {
  return React.createElement(PaletteContext.Provider, {value: defaultPalette}, children)
}

export function useAnpanTheme(): Palette {
  return useContext(PaletteContext)
}

import { useEffect } from 'react';
import { useBooleanLocalStorage } from './useLocalStorage';

const DARK_MODE_KEY = 'isDarkMode';

type Theme = 'light' | 'dark';

export type UseThemeReturn = {
    theme: Theme;
    isDarkMode: boolean;
    toggle: () => void;
};

/**
 * Hook to manage theme state and apply theme classes to the document.
 * Persists theme preference to localStorage and defaults to system preference.
 */
export function useTheme(): UseThemeReturn {
    const userPrefersDarkMode =
        window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches;
    const [isDarkMode, setIsDarkMode] = useBooleanLocalStorage(DARK_MODE_KEY, userPrefersDarkMode);

    const theme: Theme = isDarkMode ? 'dark' : 'light';

    useEffect(() => {
        const htmlElement = document.documentElement;

        // Apply PatternFly theme class (only dark mode has a class)
        if (isDarkMode) {
            htmlElement.classList.add('pf-v6-theme-dark');
        } else {
            htmlElement.classList.remove('pf-v6-theme-dark');
        }

        // Apply Tailwind theme class
        htmlElement.classList.remove('theme-light', 'theme-dark');
        htmlElement.classList.add(`theme-${theme}`);
    }, [isDarkMode, theme]);

    const toggle = () => {
        setIsDarkMode((prev) => !prev);
    };

    return {
        theme,
        isDarkMode,
        toggle,
    };
}

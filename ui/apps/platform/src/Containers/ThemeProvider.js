import React, { createContext, useContext, useState, useEffect } from 'react';
import PropTypes from 'prop-types';
import { useMediaQuery } from 'react-responsive';

const defaultContextData = {
    isDarkMode: false,
    toggle: () => {},
};

export const ThemeContext = createContext(defaultContextData);
const useTheme = () => useContext(ThemeContext);

const DARK_MODE_KEY = 'isDarkMode';

// custom react hook to toggle dark mode across UI
const useEffectDarkMode = () => {
    const userPrefersDarkMode = useMediaQuery({ query: '(prefers-color-scheme: dark)' });
    const [themeState, setThemeState] = useState({
        isDarkMode: userPrefersDarkMode,
        hasThemeMounted: false,
    });
    useEffect(() => {
        const darkModeValue = localStorage.getItem(DARK_MODE_KEY);
        let isDarkMode;
        // In the very beginning, default to using what the user prefers.
        if (darkModeValue === null) {
            isDarkMode = userPrefersDarkMode;
        } else {
            // It's always either 'true' or 'false', but if it's something unexpected,
            // default to light mode.
            isDarkMode = darkModeValue === 'true';
        }
        setThemeState({ isDarkMode, hasThemeMounted: true });
    }, [userPrefersDarkMode]);

    return [themeState, setThemeState];
};

const ThemeProvider = ({ children }) => {
    const [themeState, setThemeState] = useEffectDarkMode();

    // to prevent theme flicker while getting theme from localStorage
    if (!themeState.hasThemeMounted) {
        return <div />;
    }

    // Note: Once the app has been fully migrated to PatternFly the `theme-light` and
    // `theme-dark` classes can be removed
    const getThemeClasses = (isDarkMode) =>
        isDarkMode ? ['theme-dark', 'pf-theme-dark'] : ['theme-light'];
    document.documentElement.classList.add(...getThemeClasses(themeState.isDarkMode));
    document.documentElement.classList.remove(...getThemeClasses(!themeState.isDarkMode));

    const toggle = () => {
        const prevTheme = getThemeClasses(themeState.isDarkMode);
        const darkModeToggled = !themeState.isDarkMode;
        localStorage.setItem(DARK_MODE_KEY, JSON.stringify(darkModeToggled));
        document.documentElement.classList.remove(...prevTheme);
        setThemeState({ ...themeState, isDarkMode: darkModeToggled });
        const newTheme = getThemeClasses(darkModeToggled);

        document.documentElement.classList.add(...newTheme);
    };

    return (
        <ThemeContext.Provider
            value={{
                isDarkMode: themeState.isDarkMode,
                toggle,
            }}
        >
            {children}
        </ThemeContext.Provider>
    );
};

ThemeProvider.propTypes = {
    children: PropTypes.node.isRequired,
};

export { ThemeProvider, useTheme };

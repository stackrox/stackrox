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

        // TODO: remove this override for never dark-mode, after we update to use PatternFly themes for dark mode
        isDarkMode = false;

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

    const getTheme = (isDarkMode) => (isDarkMode ? 'theme-dark' : 'theme-light');
    document.body.classList.add(getTheme(themeState.isDarkMode));
    document.body.classList.remove(getTheme(!themeState.isDarkMode));

    const toggle = () => {
        const prevTheme = getTheme(themeState.isDarkMode);

        // TODO: remove this override for never dark-mode, ` && false`
        //       after we update to use PatternFly themes for dark mode

        const darkModeToggled = !themeState.isDarkMode && false;

        localStorage.setItem(DARK_MODE_KEY, JSON.stringify(darkModeToggled));
        document.body.classList.remove(prevTheme);
        setThemeState({ ...themeState, isDarkMode: darkModeToggled });
        const newTheme = getTheme(darkModeToggled);

        document.body.classList.add(newTheme);
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

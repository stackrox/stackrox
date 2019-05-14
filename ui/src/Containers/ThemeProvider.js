import React, { createContext, useContext, useState, useEffect } from 'react';
import PropTypes from 'prop-types';

const defaultContextData = {
    isDarkMode: false,
    toggle: () => {}
};

const ThemeContext = createContext(defaultContextData);
const useTheme = () => useContext(ThemeContext);

// custom react hook to toggle dark mode across UI
const useEffectDarkMode = () => {
    const [themeState, setThemeState] = useState({
        isDarkMode: false,
        hasThemeMounted: false
    });
    useEffect(() => {
        const isDarkMode = localStorage.getItem('isDarkMode') === 'true';
        setThemeState({ isDarkMode, hasThemeMounted: true });
    }, []);

    return [themeState, setThemeState];
};

const ThemeProvider = ({ children }) => {
    const [themeState, setThemeState] = useEffectDarkMode();

    // to prevent theme flicker while getting theme from localStorage
    if (!themeState.hasThemeMounted) {
        return <div />;
    }

    const getTheme = isDarkMode => (isDarkMode ? 'theme-dark' : 'theme-light');
    const curTheme = getTheme(themeState.isDarkMode);
    document.body.classList.add(curTheme);

    const toggle = () => {
        const prevTheme = getTheme(themeState.isDarkMode);
        const darkModeToggled = !themeState.isDarkMode;
        localStorage.setItem('isDarkMode', JSON.stringify(darkModeToggled));
        document.body.classList.remove(prevTheme);
        setThemeState({ ...themeState, isDarkMode: darkModeToggled });
        const newTheme = getTheme(darkModeToggled);
        document.body.classList.add(newTheme);
    };

    return (
        <ThemeContext.Provider
            value={{
                isDarkMode: themeState.isDarkMode,
                toggle
            }}
        >
            {children}
        </ThemeContext.Provider>
    );
};

ThemeProvider.propTypes = {
    children: PropTypes.node.isRequired
};

export { ThemeProvider, useTheme };

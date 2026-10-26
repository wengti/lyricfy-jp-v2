"use client";

import { createContext, useContext, useSyncExternalStore } from "react";

interface ThemeContextValue {
	isDark: boolean;
	toggleTheme: () => void;
}

const ThemeContext = createContext<ThemeContextValue>({ isDark: false, toggleTheme: () => {} });

// The `dark` class on <html> is the source of truth. The inline script in the root
// layout sets it before first paint; this store just watches it.
// subscribe() is to be used as a callback in useSyncExternalStore where it automatically feeds the onChange parameter
// once it has observed a change in what it is observing, onChange does all the work (which is built in useSyncExternalStore)
// where it will rerender ThemeProvider (since useSyncExternalStore returns an external state)
function subscribe(onChange: () => void) {
	const observer = new MutationObserver(onChange); // mutationObserver is provided by the browser
	observer.observe(document.documentElement, { attributes: true, attributeFilter: ["class"] }); // observe the class of <html> class
	return () => observer.disconnect();
}

const getSnapshot = () => document.documentElement.classList.contains("dark");
const getServerSnapshot = () => false;

export function ThemeProvider({ children }: { children: React.ReactNode }) {

    // subscribe - observe if the class value on the current <html> changed
    // getSnapshot - get current dark value attribute on <html>
    // getServerSnapshot - get current dark value attribute on <html> when its on server side (but no html file, so a defalt value is returned)
	const isDark = useSyncExternalStore(subscribe, getSnapshot, getServerSnapshot);

    // a function that becomes part of the context
    // so when this function is called, the site knows what to render next based on current tracked state
	function toggleTheme() {
		const next = !isDark;
		document.documentElement.classList.toggle("dark", next);
		localStorage.setItem("theme", next ? "dark" : "light");
	}

	return <ThemeContext value={{ isDark, toggleTheme }}>{children}</ThemeContext>;
}

export function useTheme() {
	return useContext(ThemeContext);
}

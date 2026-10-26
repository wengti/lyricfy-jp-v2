// Adds DOM matchers such as toBeInTheDocument(), toBeDisabled(), toHaveTextContent()
import "@testing-library/jest-dom/vitest";
import { cleanup } from "@testing-library/react";
import { afterEach } from "vitest";

afterEach(() => {
	// Unmount whatever the test rendered, and reset browser state shared between tests
	cleanup();
	if (typeof document !== "undefined") {
		document.documentElement.className = "";
		localStorage.clear();
	}
});

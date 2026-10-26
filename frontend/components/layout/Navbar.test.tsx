import { describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { usePathname } from "next/navigation";
import Navbar from "./Navbar";
import { ThemeProvider } from "@/contexts/ThemeContext";
import { logout } from "@/app/actions/auth";

vi.mock("next/navigation", () => ({ usePathname: vi.fn(() => "/dictionary") }));
// The real logout needs a live request (cookies); here we only check that the button calls it
vi.mock("@/app/actions/auth", () => ({ logout: vi.fn() }));

function renderNavbar() {
	return render(
		<ThemeProvider>
			<Navbar />
		</ThemeProvider>,
	);
}

describe("Navbar", () => {
	describe("navigation", () => {
		it("links the wordmark and the Dictionary item to /dictionary", () => {
			renderNavbar();

			expect(screen.getByRole("link", { name: "LyricfyJP" })).toHaveAttribute("href", "/dictionary");
			expect(screen.getByRole("link", { name: "Dictionary" })).toHaveAttribute("href", "/dictionary");
		});

		it("marks Dictionary as the current page while on it", () => {
			renderNavbar();
			expect(screen.getByRole("link", { name: "Dictionary" })).toHaveAttribute("aria-current", "page");
		});

		it("doesn't mark Dictionary as current on other pages", () => {
			vi.mocked(usePathname).mockReturnValueOnce("/flashcards");
			renderNavbar();
			expect(screen.getByRole("link", { name: "Dictionary" })).not.toHaveAttribute("aria-current");
		});
	});

	describe("theme toggle", () => {
		it("offers dark mode when the page is light", () => {
			renderNavbar();
			expect(screen.getByRole("button", { name: "Switch to dark mode" })).toBeInTheDocument();
		});

		it("offers light mode when the page is already dark", () => {
			// What the inline script in the root layout does for a returning dark-mode user
			document.documentElement.classList.add("dark");
			renderNavbar();
			expect(screen.getByRole("button", { name: "Switch to light mode" })).toBeInTheDocument();
		});

		it("switches to dark mode and remembers the choice", async () => {
			const user = userEvent.setup();
			renderNavbar();

			await user.click(screen.getByRole("button", { name: "Switch to dark mode" }));

			expect(document.documentElement).toHaveClass("dark");
			expect(localStorage.getItem("theme")).toBe("dark");
			expect(await screen.findByRole("button", { name: "Switch to light mode" })).toBeInTheDocument();
		});

		it("switches back to light mode", async () => {
			const user = userEvent.setup();
			document.documentElement.classList.add("dark");
			renderNavbar();

			await user.click(screen.getByRole("button", { name: "Switch to light mode" }));

			expect(document.documentElement).not.toHaveClass("dark");
			expect(localStorage.getItem("theme")).toBe("light");
			expect(await screen.findByRole("button", { name: "Switch to dark mode" })).toBeInTheDocument();
		});
	});

	describe("logout", () => {
		it("calls the logout action when the button is clicked", async () => {
			const user = userEvent.setup();
			renderNavbar();

			await user.click(screen.getByRole("button", { name: "Log out" }));

			expect(logout).toHaveBeenCalled();
		});
	});
});

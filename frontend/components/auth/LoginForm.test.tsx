import { describe, expect, it, vi } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import LoginForm from "./LoginForm";
import { login, type LoginState } from "@/app/actions/auth";

// The real action calls the backend; the form only needs something that returns a LoginState.
// Each test decides what the action answers.
vi.mock("@/app/actions/auth", () => ({
	login: vi.fn(async (): Promise<LoginState> => ({ error: null })),
}));

async function fillAndSubmit(email: string, password: string) {
	const user = userEvent.setup();
	await user.type(screen.getByLabelText("Email"), email);
	await user.type(screen.getByLabelText("Password"), password);
	await user.click(screen.getByRole("button", { name: "Sign in" }));
}

describe("LoginForm", () => {
	it("has labelled email and password fields and a sign-in button", () => {
		render(<LoginForm />);

		expect(screen.getByLabelText("Email")).toHaveAttribute("type", "email");
		expect(screen.getByLabelText("Password")).toHaveAttribute("type", "password");
		expect(screen.getByRole("button", { name: "Sign in" })).toBeEnabled();
	});

	it("shows no error before the form is submitted", () => {
		render(<LoginForm />);
		expect(screen.queryByRole("alert")).not.toBeInTheDocument();
	});

	it("shows the error returned by the login action", async () => {
		vi.mocked(login).mockResolvedValueOnce({ error: "Invalid email or password." });
		render(<LoginForm />);

		await fillAndSubmit("me@example.com", "secret");

		expect(await screen.findByRole("alert")).toHaveTextContent("Invalid email or password.");
	});

	it("sends the typed email and password to the login action", async () => {
		render(<LoginForm />);

		await fillAndSubmit("me@example.com", "secret");

		// Wait for what this test is about: the action being called
		await waitFor(() => expect(login).toHaveBeenCalledTimes(1));
		const formData = vi.mocked(login).mock.calls[0][1];
		expect(formData.get("email")).toBe("me@example.com");
		expect(formData.get("password")).toBe("secret");
	});

	it("disables the button and shows progress while signing in", async () => {
        // finish is set as a placeholder first
		let finish!: (state: LoginState) => void;

        // next time when login is triggered, we assign the resolve to finsih
        // so we get to decide when the login process is finished
		vi.mocked(login).mockImplementationOnce(() => new Promise((resolve) => (finish = resolve)));
		render(<LoginForm />);

        // trigger the login process
		await fillAndSubmit("me@example.com", "secret");

        // check if when login is pressed, the button is disabled
		const button = await screen.findByRole("button", { name: "Signing in…" });
		expect(button).toBeDisabled();

        // finish the login process
		finish({ error: "Wrong password" });

        // check if the desired outcome is reached after login process is completed
		expect(await screen.findByRole("alert")).toHaveTextContent("Wrong password");
		expect(screen.getByRole("button", { name: "Sign in" })).toBeEnabled();
	});
});

import { defineStore } from "pinia";

export interface State {
	authToken: string | null;
	faPro: boolean;
}

export const useStore = defineStore("main", {
	state: () => ({
		authToken: null,
		faPro: import.meta.env.VITE_APP_FA_PRO === "true",
	}),
});

<template>
	<nav :transparent="false">
		<router-link class="app-title unstyled-link" to="/">
			<div class="logo">
				<Logo />
			</div>

			<div class="text">
				<span class="name">7tv.io</span>
				<span class="dev-stage-text">dev portal</span>
			</div>
		</router-link>

		<button class="toggle-collapse" @click="toggleNav">Menu</button>
		<div class="collapse">
			<div class="nav-links">
				<div v-for="link of navLinks" :key="link.route">
					<router-link v-if="!link.condition || link.condition()" class="nav-link" :to="link.route">
						<span :style="{ color: link.color }">{{ link.label.toUpperCase() }}</span>
					</router-link>
				</div>
			</div>

			<div class="account"></div>
		</div>

		<span v-if="env !== 'production'" class="env">
			{{ env.toString().toUpperCase() }}
		</span>
	</nav>
	<div class="realNav" style="margin-top: 4.5rem" v-if="navOpen">
		<div>
			<div v-for="link of navLinks" :key="link.route">
				<router-link v-if="!link.condition || link.condition()" class="nav-link" :to="link.route">
					<span :style="{ color: link.color }">{{ link.label }}</span>
				</router-link>
			</div>
		</div>
	</div>
</template>

<script setup lang="ts">
import { ref, watch } from "vue";
import Logo from "@svg/Logo.vue";
import { useRoute } from "vue-router";

const route = useRoute();

const env = import.meta.env.VITE_APP_ENV;

const navOpen = ref(false);

const toggleNav = () => {
	navOpen.value = !navOpen.value;
};

const navLinks = ref([
	{ label: "Intro", route: "/" },
	{ label: "Docs", route: "/docs" },
	{ label: "Apps", route: "/apps" },
] as NavLink[]);

interface NavLink {
	label: string;
	route: string;
	color?: string;
	condition?: () => boolean;
}

watch(route, () => {
	navOpen.value = false;
});
</script>

<style scoped lang="scss">
@import "@style/themes.scss";

nav {
	position: fixed;
	z-index: 100;
	top: 0;
	width: 100%;
	height: 4.5rem;
	max-height: 4.5rem;
	min-height: 4.5rem;
	padding: 0 1.5vw;
	display: flex;
	font-size: 1.25rem;
	transition: background-color 100ms ease-in;
	backdrop-filter: blur(15px);

	@include themify() {
		background-color: themed("navBackgroundColor");
	}
}

.app-title {
	--logo-primary: currentColor;
	--logo-secondary: #4fc2bc;
	z-index: 1;
	display: flex;
	align-items: center;

	&:hover {
		color: inherit;
	}

	.logo {
		display: flex;
		color: #e12d2d;
		width: 1em;
		height: 1em;
		margin-right: 0.25em;
		font-size: 1.25em;
		background-repeat: no-repeat;
		background-size: cover;
	}

	.text .name {
		font-weight: 700;
		font-family: "Work Sans", sans-serif;
		font-size: 1.25em;
	}

	.dev-stage-text {
		width: 0;
		position: relative;
		font-size: 0.5em;
		bottom: -1em;
		left: -50%;
		font-family: "Work Sans", sans-serif;
		font-weight: 900;
		color: #e12d2d;

		&.env-offset {
			bottom: 1em;
			left: -100%;
			font-weight: 600;
		}
	}
}

.account {
	display: flex;
	flex-wrap: wrap;
	width: 20%;
}

.env {
	position: absolute;
	display: flex;
	width: 100%;
	height: 100%;
	justify-content: center;
	align-items: center;
	pointer-events: none;
	user-select: none;
	letter-spacing: 1em;
	font-size: 3em;

	@include themify() {
		color: transparentize(themed("color"), 0.85);
	}
}

.nav-links {
	display: flex;
	justify-content: center;
	grid-gap: 0.25em;
	flex-grow: 1;
	width: 0;
}

.nav-link {
	color: inherit;
	text-decoration: none;
	display: grid;
	place-items: center;
	padding: 1em;
	height: 100%;
	border-bottom: 0.1em solid transparent;
	font-size: 0.85em;
	transition: all 0.2s ease-in-out;

	@include breakpoint(md, max) {
		&.router-link-active {
			background-color: #ca0000b4;
		}

		&:hover {
			@include themify() {
				background: #e4000038;
			}
		}
	}

	@include breakpoint(lg, min) {
		&.router-link-active {
			border-color: currentColor;
		}

		&:hover {
			@include themify() {
				background: transparentize(themed("backgroundColor"), 0.5);
			}
		}
	}
}

.collapse {
	display: flex;
	z-index: 1000;
	flex-grow: 1;
}

.toggle-collapse {
	display: none;
	background-color: transparent;
	font: inherit;
	color: inherit;
	border: 0.1em solid transparent;
	padding: 0.5em;
	border-radius: 0.5em;
	place-self: center;

	&:hover {
		border-color: #303030;
	}

	&:active {
		background-color: #424242;
	}
}

@media screen and (max-width: 1000px) {
	.realNav {
		@include breakpoint(lg, min) {
			display: none;
		}

		transition: all 0.5s ease-in-out;
		font-size: 1.5rem;
		display: block;
		margin-top: 4.5rem;
		z-index: 150;
		position: fixed;
		width: 100%;
		box-shadow: black 0.5rem 0 15rem;
		backdrop-filter: blur(1rem);

		@include themify() {
			background-color: transparentize(themed("extreme"), 0.33);
		}
	}

	.navOpen {
		.collapse {
			min-height: calc(100vh - 4.5rem);
			min-width: 100vw;
			top: 4.5rem;
			left: 0;
			backdrop-filter: blur(0.75em) grayscale(30%) brightness(70%);

			@include themify() {
				background-color: transparentize(themed("extreme"), 0.33);
			}

			position: absolute;
			z-index: 1000;
			flex-direction: column;

			.account {
				width: 100vw;
				place-items: center;
				justify-content: center;
				font-size: 1.5em;
			}

			.nav-links {
				flex-direction: column;
				justify-content: start;
				width: 100vw;
				order: 2;

				.nav-link {
					&.router-link-exact-active {
						border-color: transparent !important;

						@include themify() {
							background: mix(themed("backgroundColor"), #6b6b6b, 75%);
						}
					}
				}
			}
		}

		.account {
			position: unset;
		}

		.nav-link {
			width: 100vw;
		}
	}

	nav:not(.navOpen) {
		.collapse {
			display: none;
		}
	}

	.toggle-collapse {
		margin-left: auto;
		display: block;
	}
}
</style>

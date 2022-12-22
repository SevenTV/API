<template>
	<div id="sidebarmain">
		<div id="sidebar" class="docs-sidebar">
			<!-- Version Selector -->
			<div class="sidebar-version-selector">
				<p class="sidebar-section">API HOST</p>
				<span>
					<span>{{ host.replace(/^(https?:|)\/\//, "") }}/</span>
					<select v-model="version">
						<option v-for="v of versions">{{ v }}</option>
					</select>
				</span>
				<button class="toggle-collapse" @click="toggleNav">Expand Nav</button>
			</div>
			<div v-if="routeLinkNavOpen" class="sidebar-routes-big">
				<p class="sidebar-section">ROUTES</p>
				<div class="route-navigator">
					<div class="route-navigator-item" v-for="route in routes" :key="route.id">
						<a class="route-navigator-link" @click="scrollTo($event, route.id)">{{ route.summary }}</a>
					</div>
				</div>
			</div>
			<div id="routeNavigationToggle" class="sidebar-routes-small">
				<p class="sidebar-section">ROUTES</p>
				<div class="route-navigator">
					<div class="route-navigator-item" v-for="route in routes" :key="route.id">
						<a class="route-navigator-link" @click="scrollTo($event, route.id)">{{ route.summary }}</a>
					</div>
				</div>
			</div>
		</div>
	</div>
</template>

<script setup lang="ts">

import { ref } from "vue";

const host = import.meta.env.VITE_APP_API_REST ?? "";
const versions = ref<string[]>(["v3"]);
const version = ref("v3");

const routeLinkNavOpen = ref(false);

const toggleNav = () => {
	routeLinkNavOpen.value = !routeLinkNavOpen.value;
};

const scrollTo = (evt: MouseEvent, id: string) => {
	const el = document.getElementById(id);
	if (!el) return;

	el.scrollIntoView({ behavior: "smooth" });
};

defineProps<{
	host: string;
	versions: object;
	routes: any;
}>();

</script>

<style scoped lang="scss">

@import "@style/themes.scss";

#sidebarmain {
	display: grid;
	scroll-behavior: smooth;
	position: sticky;
	top: 4.5rem;

	.toggle-collapse {
		display: none;
		background-color: #373737;
		font: inherit;
		color: inherit;
		margin-left: 0.313em;
		border: 0.1em solid transparent;
		padding: 0.3em;
		border-radius: 0.5em;
		place-self: center;

		&:hover {
			border-color: #373737;
		}

		&:active {
			background-color: #515151;
		}
	}

	.docs-sidebar {
		display: flex;
		flex-direction: column;
		background-color: red;
		padding-left: 1em;
		padding-top: 1em;
		gap: 1em;
		max-height: calc(100vh - 4.5rem);
		overflow: auto;

		.sidebar-routes-big {
			// this is just to keep it hidden on default for smaller screens. JS will hide/unhide it.
			@include breakpoint(md, min) {
				display: none;
			}
		}

		.sidebar-routes-small {
			// this is just to keep it hidden on default for smaller screens. JS will hide/unhide it.
			@include breakpoint(md, max) {
				display: none;
			}
		}

		.sidebar-section {
			font-size: 0.85rem;
			font-weight: 600;
		}

		> .sidebar-version-selector > select {
			margin-top: 0.5em;
		}
	}

	@include themify() {
		.route-navigator-link {
			cursor: pointer;
			display: block;
			padding: 0.625rem;
			margin: 0.313em 0.938em 0.313em 0;
			color: themed("color");
			background-color: themed("backgroundColor");

			.route-navigator-link:hover {
				background-color: themed("secondaryBackgroundColor");
			}
		}
	}

	.route-navigator-item {
		transition: all 0.1s ease-in-out;
		border: red;

		:hover {
			border-left: 0.25rem solid red;
			border-bottom: 0.25rem solid red;
			transition: all 0.1s ease-in-out;
		}
	}

	@include breakpoint(md, max) {
		display: block;

		.docs-sidebar {
			padding-bottom: 1rem;
		}

		.toggle-collapse {
			display: inline;
		}
	}

	@include themify() {
		.docs-sidebar {
			position: sticky;
			top: 4.5rem;
			background-color: lighten(themed("backgroundColor"), 2);
		}
	}
}

</style>

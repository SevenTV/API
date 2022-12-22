<template>
	<Nav />

	<main class="entrypoint">
		<router-view v-slot="{ Component: routeComponent, route: usedRoute }" class="bouncy">
			<Transition
				:name="(usedRoute.meta.transition as string)"
				:mode="usedRoute.meta.transitionMode as unknown as any ?? 'default'"
			>
				<Suspense>
					<component :is="routeComponent" />
				</Suspense>
			</Transition>
		</router-view>
	</main>
</template>

<script setup lang="ts">

import { useHead } from "@vueuse/head";
import Nav from "./components/Nav.vue";

useHead({
	bodyAttrs: {
		class: "theme-dark",
	},
});

</script>

<style></style>

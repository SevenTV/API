<template>
	<font-awesome-icon :size="size" :icon="[lib ?? (pro ? 'far' : 'fas'), icon]" />
</template>

<script setup lang="ts">

import { useStore } from "@/store/main";
import { storeToRefs } from "pinia";

defineProps<{
	icon: string;
	lib?: string;
	size?: Size;
}>();

const { faPro: pro } = storeToRefs(useStore());

type Size = "xs" | "sm" | "md" | "lg" | "xl";

</script>

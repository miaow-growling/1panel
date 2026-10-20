<template>
    <RouterViewCache :keep-alive="keepAlive" />
</template>

<script setup lang="ts">
import type { KeepAlivePattern } from '@/utils/tab-cache';

defineProps<{
    keepAlive?: KeepAlivePattern | null;
}>();
</script>

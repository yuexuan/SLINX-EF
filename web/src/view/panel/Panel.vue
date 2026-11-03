<template>
    <Nav />
    <div class="view">
        <header class="panel-header"><ThemeToggle /></header>
        <router-view />
    </div>
</template>

<script setup lang="ts">
import Nav from './Nav.vue'
import ThemeToggle from '@/component/ThemeToggle.vue'
</script>
<style scoped>
.panel-header { display: flex; justify-content: flex-end; padding: 12px 20px 0; flex-shrink: 0; }
</style>

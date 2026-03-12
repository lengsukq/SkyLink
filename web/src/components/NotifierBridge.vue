<template />

<script setup>
import { onMounted } from 'vue'
import { useNotification } from 'naive-ui'
import { setNotifier } from '../ui/notify'

const notification = useNotification()

onMounted(() => {
  setNotifier(notification)
})
</script>


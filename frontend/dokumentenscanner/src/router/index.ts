import { createRouter, createWebHashHistory } from 'vue-router'

const router = createRouter({
  history: createWebHashHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/',
      redirect: '/desktop',
    },
    {
      path: '/desktop',
      name: 'desktop',
      component: () => import('../views/DesktopView.vue'),
    },
    {
      path: '/mobile',
      name: 'mobile',
      component: () => import('../views/MobileView.vue'),
    },
    {
      path: '/:pathMatch(.*)*',
      redirect: '/desktop',
    },
  ],
})

export default router

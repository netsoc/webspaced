const router = new VueRouter({
  mode: 'history',
  routes: [
    { path: '/', component: Vue.component('HomeView') },

    { path: '*', component: Vue.component('NotFound') }
  ]
});

const vm = new Vue({
  el: '#app',
  router
});

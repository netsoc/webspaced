const router = new VueRouter({
    mode: 'history',
    routes: [{
            path: '/',
            component: Vue.component('HomeView')
        },
        {
            path: '/login',
            component: Vue.component('Login')
        },
        {
            path: '/dashboard',
            component: Vue.component('Dashboard')
        },
        {
            path: '/welcome',
            component: Vue.component('Welcome')
        },
        {
            path: '/choose-os',
            component: Vue.component('Operating System')
        },
        {
            path: '*',
            component: Vue.component('NotFound')
        },
    ]
});

const vm = new Vue({
    el: '#app',
    router,
});
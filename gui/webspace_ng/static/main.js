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
            path: '/Terminal',
            component: Vue.component('Terminal')
        },
        {
            path: '/configs',
            component: Vue.component('Configs')
        },
        {
            path: '/domains',
            component: Vue.component('Domains')
        },
        {
            path: '/ports',
            component: Vue.component('Ports')
        },
        {
            path: '/welcome',
            component: Vue.component('Welcome')
        },
        {
            path: '/choose-os',
            component: Vue.component('OperatingSystem')
        },
        {
            path: '/create-root',
            component: Vue.component('CreateRootPW')
        },
        {
            path: '/congrats',
            component: Vue.component('Congratulations')
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

Vue.config.delimiters = ["${", "}"];
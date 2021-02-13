import Vue from 'vue'
import VueRouter from 'vue-router'
import VueApexCharts from 'vue-apexcharts'
import VueObserveVisibility from 'vue-observe-visibility'
import VueClipboard from 'vue-clipboard2'
import VueCookies from 'vue-cookies'
import VueI18n from 'vue-i18n'
import router from './routes'
import "./mixin"
import "./icons"
import store from './store'
import language from './languages'

import { BootstrapVue} from 'bootstrap-vue'
// Import Bootstrap an BootstrapVue CSS files (order is important)
//import 'bootstrap/dist/css/bootstrap.css'
//import 'bootstrap-vue/dist/bootstrap-vue.css'

const App = () => import(/* webpackChunkName: "index" */ '@/App.vue')

Vue.component('apexchart', VueApexCharts)

// Make BootstrapVue available throughout your project
Vue.use(BootstrapVue)

Vue.use(VueClipboard);
Vue.use(VueRouter);
Vue.use(VueObserveVisibility);
Vue.use(VueCookies);
Vue.use(VueI18n);

const i18n = new VueI18n({
  fallbackLocale: "en",
  messages: language
});

Vue.$cookies.config('3d')

Vue.config.productionTip = process.env.NODE_ENV !== 'production'
Vue.config.devtools = process.env.NODE_ENV !== 'production'
Vue.config.performance = process.env.NODE_ENV !== 'production'

new Vue({
  router,
  store,
  i18n,
  render: h => h(App),
}).$mount('#app')

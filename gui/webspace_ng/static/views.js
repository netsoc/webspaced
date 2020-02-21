Vue.component('NotFound', {
  template: `
    <div>
      <h1>Not Found</h1>
      <p class="lead">Couldn't find that page, sorry.</p>
      <code>¯\\_(ツ)_/¯</code>
    </div>
  `
});

Vue.component('HomeView', {
  template: `
    <div>
      <h1>Home</h1>
      <p class="lead">Hello, world!</p>
    </div>
  `
});

Vue.component('Login', {
  template: `
    <div>
      <form action="/post" v-if="!loading">
        <label><b>Username</b></label>
        <input type="text" v-model="username">
        <label><b>Password</b></label>
        <input type="password" v-model="password">
        <input type="submit" v-on:click.prevent="login">
      </form>
      <Loading v-if="loading"></Loading>
    </div>
  `
});

Vue.component('NotFound', {
  template: `
    <div>
      <h1>Not Found</h1>
      <p class="lead">Couldn't find that page, sorry.</p>
      <code>¯\\_(ツ)_/¯</code>
    </div>
  `
});
1
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
  <template>
    <div class ="center">
      <img class="center" src="/static/images/logo.jpg" alt="Netsoc Logo">
      <form @submit.prevent="handleSubmit">
          <div class="form-group login-box center">
              <input type="text" name="username" class="form-control" placeholder="Username" style="border:none"/>
          </div>
          <div class="form-group login-box center">
              <input type="password" name="password" class="form-control" placeholder="Password" style="border:none"/>
          </div>
      </form>
    </div>
  </template>
  `
});

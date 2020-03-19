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

Vue.component('Navbar', {
    template: `
    <div>
      <div class="sidenav">
        <i class="fa fa-home fa-lg"> <a href="/dashboard"> Dashboard </a></i>
        <i class="fa fa-terminal fa-lg"> <a href="/terminal"> Terminal </a></i>
        <i class="fa fa-cog fa-lg"> <a href="/configs"> Configs </a></i>
        <i class="fa fa-globe fa-lg"> <a href="/domains"> Domains </a></i>
        <i class="fa fa-plug fa-lg"> <a href="/ports"> Ports </a></i>
      </div>
    </div>
  `
});

Vue.component('Graph', {
  template: `
    <body>
      <div class="container">
        <canvas id ="myChart></canvas>
      </div>
      <script>
      </script>
    </body>
  `
});

Vue.component('Dashboard', {
  template:` 
  <div>
    <p> Dashboard </p>
    <navbar></navbar>
  </div>
`
});

Vue.component('Terminal', {
  template:` 
  <div>
    <navbar></navbar>
  </div>
`
});

Vue.component('Configs', {
  template:` 
  <div>
    <div>
      <h2>HTTP/HTTPS Ports</h2>
      <input type="text" placeholder="HTTP Port">
      <br>
      <input type="text" placeholder="HTTPs Port">
      <br>
      <h2>Startup Delay (Seconds)</h2>
      <input type="text" placeholder="Delay">
      <br>
      <h2> Enable SSL Termination </h2>
      <label class="switch">
        <input type="checkbox">
        <span class="slider round"></span>
      </label>
    </div> 
    <navbar></navbar>
  </div>
`
});

Vue.component('Domains', {
  template:` 
  <div>
    <h2>Domains</h2>
    <input type="text" placeholder="Domains">
    <navbar></navbar>
  </div>
`
});

Vue.component('Ports', {
  template:` 
  <div>
    <h2>External Ports</h2>
    <input type="text" placeholder="External Port">
    <br>
    <input type="text" placeholder="External Port">
    <br>
    <h2>Internal Ports</h2>
    <input type="text" placeholder="Internal Port">
    <br>
    <input type="text" placeholder="Internal Port">
    <br>
    <navbar></navbar>
  </div>
`
});


Vue.component('Operating System', {
    template: `
    <div class = "container">
      <div class = "row">
        <div class = "col"> Ubuntu </div> 
        <div class = "col"> Arch </div> 
        <div class = "col"> Redfin </div> 
      </div> 
      <br>
      <div class = "row">
        <div class = "col"> Fedora </div> 
        <div class = "col"> Centos </div> 
        <div class = "col"> Alpine </div> 
      </div>
    </div> 
    <button class = "center"type = "button"> Next </button> 
    <footer style = "text-align: center"> Brought to you by DU Netsoc</footer>
  `
});

Vue.component('Login', {
    template: `
    <template>
      <div class ="center">
        <img class="center" id="login-logo" src="/static/images/logo.png" alt="Netsoc Logo">
        <form @submit.prevent="handleSubmit">
            <div class="form-group login-box center">
                <input type="text" name="username" class="form-control" placeholder="Username" style="border:none; background-color:#fff;"/>
            </div>
            <div class="form-group login-box center">
                <input type="password" name="password" class="form-control" placeholder="Password" style="border:none"/>
            </div>
        </form>
        <div class="bottom-right-corner">
          <a href= "/welcome" class="button center"> Login </a>
        </div>
        <div class="bottom-left-corner">
          <p>Made by DU Netsoc</p>
        </div>
      </div>
    `
});

Vue.component('Welcome', {
    template: `
    <div> 
      <h1 style = "text-align: center"> Welcome to your dashboard </h1> 
      <br>
      <h4 style= "text-align: center"> Setup your environment to use your webspace </h4> 
      <br>
      <br>
      <br>
      <a class="button center" href="/choose-os"> Get Started </a>
      <div class="bottom-left-corner">
        <p>Made by DU Netsoc</p>
      </div>
    <div>
  `
});

Vue.component('Operating System', {
    template: `
  <div>
  <h1 style="text-align: center;">Choose your Operating System</h1>
  <br>
    <div class = "container" >
      <div class = "row">
        <div class = "col"> 
        <h4 style = "text-align: center">Arch</h4>  
        <img src="/static/images/Arch.png" alt="Arch Logo" class="center-img">
          <button class="center button3"> Select </download>
        </div>

        <div class = "col"> 
          <h4 style = "text-align: center">Alpine</h4>
          <img src="/static/images/Alpine.png" alt="Alpine Logo" class="center-img">
          <button class="center button3"> Select </download>
        </div>

        <div class = "col"> 
          <h4 style = "text-align: center">Centos</h4> 
          <img src="/static/images/Centos.png" alt="Centos Logo" class="center-img">
          <button class="center button3"> Select </download>
        </div> 
      </div> 
      <br>

      <div style="margin-top:2em;" class = "row">
        <div class = "col">
          <h4 style = "text-align: center">Debian</h4> 
          <img src="/static/images/Debian.png" alt="Debian Logo" class="center-img">
          <button class="center button3"> Select </download> 
        </div> 
        
        <div class = "col">
          <h4 style = "text-align: center">Fedora</h4>  
          <img src="/static/images/Fedora.png" alt="Fedora Logo" class="center-img">
          <button class="center button3"> Select </download>
        </div> 

        <div class = "col">
          <h4 style = "text-align: center">Ubuntu</h4> 
          <img src="/static/images/Ubuntu.png" alt="Ubuntu Logo" class="center-img">
          <button class="center button3"> Select </download>
        </div> 
      </div>
    </div>
    <a class="button2 bottom-right-corner" href= "/create-root"> Next </a>    
  </div>
  `
});

Vue.component('Create Root PW', {
    template: `
  <div>
    <h1>Create your Root Password</h1>
    <input type="text" placeholder="Enter Password">
    <br>
    <input type="text" placeholder="Re-enter Password">
    <br>
    <h1 style="margin-top: 50px;">Create an SSH Key (Optional)</h1>
    <textarea></textarea>
    <a class="button2 bottom-left-corner" href= "/choose-os"> Previous </a> 
    <a class="button2 bottom-right-corner" href= "/congrats"> Next </a>
  </div> 
  `
});

Vue.component('Congratulations', {
    template: `
    <div>
      <h1 style = "text-align: center; margin-bottom:1.5em;"> You have succesfully setup your webspace!</h1>
      <img class="center" src="/static/images/icons8-ok-300.png" alt="completed">
      <a class="button2 bottom-left-corner" href= "/create-root"> Previous </a> 
      <a class="button2 bottom-right-corner" href= "/dashboard"> Finish </a>
    </div>   
  `
});
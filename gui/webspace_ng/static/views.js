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
      <img class="center" id="login-logo" src="/static/images/netsoc.png" alt="Netsoc Logo" style="width:95px;height:95px;"> 
      <h1 style = "text-align: center"> Netsoc Webspaces </h1> 
      <br>
      <br>
      <h5 style= "text-align: center">Netsoc's webspaces provide a container for every member to set up their own website.</h5>
      <br>
      <br>
      <a class="button center" href="/login"> Get Started </a>
      <div class="bottom-left-corner">
        <p>Made by DU Netsoc</p>
      </div>
    </div>
  `
});

Vue.component('Navbar', {
    template: `
    <div>
      <div class="sidenav">
        <div class="links">
          <a href="/dashboard"><i class="fa fa-home fa-lg" />  Dashboard </a>
          <a href="/terminal"><i class="fa fa-terminal fa-lg" />  Terminal </a>
          <a href="/configs"><i class="fa fa-cog fa-lg" />  Configs </a>
          <a href="/domains"><i class="fa fa-globe fa-lg" />  Domains </a>
          <a href="/ports"><i class="fa fa-plug fa-lg" />  Ports </a>
        </div>
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
    </body>
  `
});

Vue.component('Dashboard', {
    template: ` 
    <div class="main">
      <navbar></navbar>
      <img src="/static/images/Arch.png" alt="Arch Logo">
      <h2> Installed OS </h2>
      <h2 class = "graph"> CPU Usage </h2>
      <h3 class = "graph"> 2457 mb / 4096 mb  </h3>
      <div class ="graph" id="my_dataviz"></div>
      <br>
      <a class="button center" href="#"> Shut Down </a>
      <a class="button center" href="#"> Restart </a>
      <br>
    </div>  
  `,
    mounted() {
        // set the dimensions and margins of the graph
        var width = 450
        height = 450
        margin = 40

        // The radius of the pieplot is half the width or half the height (smallest one). I subtract a bit of margin.
        var radius = Math.min(width, height) / 2 - margin

        // append the svg object to the div called 'my_dataviz'
        var svg = d3.select("#my_dataviz")
            .append("svg")
            .attr("width", width)
            .attr("height", height)
            .append("g")
            .attr("transform", "translate(" + width / 2 + "," + height / 2 + ")");
          // Create dummy data
          var data = {a: 60, b: 40}

              // set the color scale
              var color = d3.scaleOrdinal()
                  .domain(data)
                  .range(["#98abc5", "#f2f2f2", "#7b6888", "#6b486b", "#a05d56"])

              // Compute the position of each group on the pie:
              var pie = d3.pie()
                  .value(function(d) { return d.value; })
              var data_ready = pie(d3.entries(data))

              // Build the pie chart: Basically, each part of the pie is a path that we build using the arc function.
              svg
                  .selectAll('whatever')
                  .data(data_ready)
                  .enter()
                  .append('path')
                  .attr('d', d3.arc()
                      .innerRadius(100) // This is the size of the donut hole
                      .outerRadius(radius)
                  )
                  .attr('fill', function(d) { return (color(d.data.key)) })
                  .attr("stroke", "black")
                  .style("stroke-width", "2px")
                  .style("opacity", 0.7)
          }
      });

Vue.component('Terminal', {
    template: ` 
  <div class="main">
    <navbar></navbar>
    <div id="terminal"></div>
  </div>
  `,
    mounted() {
        var term = new Terminal();
        term.open(document.getElementById('terminal'));
        term.write('Hello from \x1B[1;3;31mnetsoc terminal\x1B[0m $ ')
          if (term._initialized) {
              return;
          }
          term._initialized = true;
          term.writeln('This is a local terminal emulation');
          term.writeln('Type some keys and commands to play around.');
          term.onKey(e => {
              const printable = !e.domEvent.altKey && !e.domEvent.altGraphKey && !e.domEvent.ctrlKey && !e.domEvent.metaKey;
              if (e.domEvent.keyCode === 13) {
                term.write('\r\n$ ');
              } else if (e.domEvent.keyCode === 8) {
                  if (term._core.buffer.x > 2) {
                      term.write('\b \b');
                  }
              } else if (printable) {
                  term.write(e.key);
              }
          });
    }
});

Vue.component('Configs', {
    template: ` 
  <div class="main">
    <div>
      <h2>HTTP/HTTPS Ports</h2>
      <input v-model="HTTP" type="text" placeholder="HTTP Port">
      <br>
      <input v-model="HTTPS" type="text" placeholder="HTTPs Port">
      <br>
      <h2>Startup Delay (Seconds)</h2>
      <input v-model="Startup" type="text" placeholder="Delay">
      <br>
      <h2> Enable SSL Termination </h2>
      <label class="switch">
        <input v-model="SSL" type="checkbox">
        <span class="slider round"></span>
      </label>
      <div class="btn button bottom-right-corner" v-on:click="submit"> Submit Configs </div>
      <div>
        <h5 style="margin: 20px 0 20px 0;color:grey">Current configurations:</h5>
        <p style="color: grey">HTTP: {{ currentHTTP }}   
          HTTPS: {{ currentHTTPS }}   
          Startup Delay: {{ currentStartup }}s   
          SSL Termination: {{ currentSSL }}</p>
      </div>
    </div> 
    <navbar></navbar>
  </div>
`,
data: function() {
  return {
    HTTP: "",
    HTTPS: "",
    Startup: "",
    SSL: "",
    currentHTTP: "",
    currentHTTPS: "",
    currentStartup: "",
    currentSSL: ""
  }
},
mounted() {
  axios.get('/api/getConfigs')
    .then(response => {
      this.currentHTTP = response.data.HTTP
      this.currentHTTPS = response.data.HTTPS
      this.currentStartup = response.data.Startup
      this.currentSSL = response.data.SSL
    })
    .catch(error => {
      console.log(error)
    })
},
methods: {
  submit: function() {
    var configs = {"HTTP": this.HTTP, "HTTPS": this.HTTPS, "Startup": this.Startup, "SSL": this.SSL}
    $.ajax({
      type: "POST",
      contentType: "application/json;charset=utf-8",
      url: "/api/submitConfigs",
      traditional: "true",
      data: JSON.stringify({configs}),
      dataType: "json",
      success: function(response) {
        if(response.state == true) {
          alert("Submitted configs successfully")
        } else {
          alert("Didn't work")
        }
      },
      error: function(error) {
        console.log(error)
      }
    })
  } 
}
});

Vue.component('Domains', {
    template: ` 
  <div class="main">
    <h2>Domains</h2>
    <ul id="domains">
      <li id="element1"> <input v-model="domain" type="text" placeholder="Domains"> </li>
    </ul>
    <div class="btn button" v-on:click="submit" > Add Domain </div>
    <div>
      <h5 style="margin: 20px 0 10px 0; color: grey">Current Domains:</h5>
      <p style="color: grey" v-for="domain in domains">{{ domain }}</p>
    </div>
    <navbar></navbar>
  </div>
  `,
  data: function() {
    return {
      domain: "",
      domains: [""]
    }
  },
  mounted() {
    axios.get('/api/getDomains')
      .then(response => {
        this.domains = response.data.domains
      })
      .catch(error => {
        console.log(error)
      })
  },
  methods: {
    submit: function() {
      var toSubmit = {"domain": this.domain}
      $.ajax({
          type: "POST",
          contentType: "application/json;charset=utf-8",
          url: "/api/domains",
          traditional: "true",
          data: JSON.stringify({toSubmit}),
          dataType: "json",
          success: function(response) {
            if(response.result == true) {
              alert(response.domain + " successfully added")
            } else {
              alert("Error adding this domain")
            }
          },
          error: function(error) {
            console.error(error)
          }
        })
    }
  }
  
});

Vue.component('Ports', {
    template: ` 
  <div class="main">
    <h2>External Ports</h2>
    <input v-model="external1" type="text" placeholder="External Port">
    <br>
    <input v-model="external2" type="text" placeholder="External Port">
    <br>
    <h2>Internal Ports</h2>
    <input v-model="internal1" type="text" placeholder="Internal Port">
    <br>
    <input v-model="internal2" type="text" placeholder="Internal Port">
    <br>
    <navbar></navbar>
    <div class="btn button" v-on:click="submit" > Submit </div>
  </div>
`,
data: function() {
  return {
    external1: "",
    external2: "",
    internal1: "",
    internal2: ""
  }
},
methods: {
  submit: function() {
    var details = {"external1": this.external1, "external2": this.external2,
      "internal1": this.internal1, "internal2": this.internal2}
    $.ajax({
          type: "POST",
          contentType: "application/json;charset=utf-8",
          url: "/api/ports",
          traditional: "true",
          data: JSON.stringify({details}),
          dataType: "json",
          success: function(response) {
            if(response.result == true) {
              alert("Successfully added")
            } else {
              alert("Error adding these ports")
            }
          },
          error: function(error) {
            console.error(error)
          }
        })
  }
}
});


Vue.component('OperatingSystem', {
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
    <button class = "center" type = "button"> Next </button> 
    <footer style = "text-align: center"> Brought to you by DU Netsoc</footer>
  `
  
});

Vue.component('Login', {
    template: `
      <div class ="center">
        <img class="center" id="login-logo" src="/static/images/logo.png" alt="Netsoc Logo">
            <div class="form-group login-box center">
                <input type="text" v-model="email" name="email" class="form-control" placeholder="Email" style="border:none; background-color:#fff;"/>
            </div>
            <div class="form-group login-box center">
                <input type="password" v-model="password" name="password" class="form-control" placeholder="Password" style="border:none"/>
                <p>{{ error }}</p>
            </div>
        </form>
        <div class="bottom-right-corner">
          <a v-on:click="submit" class="button center"> Login </a>
        </div>
        <div class="bottom-left-corner">
          <p>Made by DU Netsoc</p>
        </div>
      </div>
    `,
    data: function() {
      return {  
        email: "",
        password: ""
      }
    },
    methods: {
      submit: function() {
        var details = {"email": this.email, "password": this.password}
        $.ajax({
          type: "POST",
          contentType: "application/json;charset=utf-8",
          url: "/api/login",
          traditional: "true",
          data: JSON.stringify({details}),
          dataType: "json",
          success: function(response) {
            if(response.state == 1) {
              window.location.href = "/welcome"
            } else if(response.state == 2) {
              window.loaction.href = "/dashboard"
            } else {
              alert("Email/Password incorrect")
            }
          },
          error: function(error) {
            console.error(error)
          }
        })
      }
    }
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

Vue.component('OperatingSystem', {
    template: `
  <div>
  <h1 style="text-align: center;">Choose your Operating System</h1>
  <br>
    <div class = "container" >
      <div class = "row">
        <div class = "col"> 
        <h4 style = "text-align: center">Arch</h4>  
        <img src="/static/images/Arch.png" alt="Arch Logo" class="center-img">
          <button v-on:click="os = 1" class="center button3"> Select </download>
        </div>

        <div class = "col"> 
          <h4 style = "text-align: center">Alpine</h4>
          <img src="/static/images/Alpine.png" alt="Alpine Logo" class="center-img">
          <button v-on:click="os = 2" class="center button3"> Select </download>
        </div>

        <div class = "col"> 
          <h4 style = "text-align: center">Centos</h4> 
          <img src="/static/images/Centos.png" alt="Centos Logo" class="center-img">
          <button v-on:click="os = 3" class="center button3"> Select </download>
        </div> 
      </div> 
      <br>

      <div style="margin-top:2em;" class = "row">
        <div class = "col">
          <h4 style = "text-align: center">Debian</h4> 
          <img src="/static/images/Debian.png" alt="Debian Logo" class="center-img">
          <button v-on:click="os = 4" class="center button3"> Select </download> 
        </div> 
        
        <div class = "col">
          <h4 style = "text-align: center">Fedora</h4>  
          <img src="/static/images/Fedora.png" alt="Fedora Logo" class="center-img">
          <button v-on:click="os = 5" class="center button3"> Select </download>
        </div> 

        <div class = "col">
          <h4 style = "text-align: center">Ubuntu</h4> 
          <img src="/static/images/Ubuntu.png" alt="Ubuntu Logo" class="center-img">
          <button v-on:click="os = 6" class="center button3"> Select </download>
        </div> 
      </div>
    </div>
    <a v-on:click="submit" class="button2 bottom-right-corner" > Next </a>    
  </div>
  `,
  data: function() {
    return {
      os : ""
    }
  },
  methods: {
      submit: function() {
        var details = {"os": this.os}
        $.ajax({
          type: "POST",
          contentType: "application/json;charset=utf-8",
          url: "/api/os",
          traditional: "true",
          data: JSON.stringify({details}),
          dataType: "json",
          success: function(response) {
            if(response.state == 1) {
              window.location.href = "/create-root"
            } else {
              alert("Error")
            }
          },
          error: function(error) {
            console.error(error)
          }
        })
      }
    }
});

Vue.component('CreateRootPW', {
    template: `
  <div>
    <h1>Create your Root Password</h1>
    <input v-model="password" type="password" class="rootpassword" placeholder="Enter Password">
    <br>
    <input v-model="confirmPassword" type="password" class="rootpassword" placeholder="Re-enter Password">
    <br>
    <h1 style="margin-top: 50px;">Create an SSH Key (Optional)</h1>
    <textarea v-model="ssh"></textarea>
    <a class="button2 bottom-left-corner" href= "/choose-os"> Previous </a> 
    <a v-on:click="submit" class="button2 bottom-right-corner"> Next </a>
  </div> 
  `,
  data : function() {
    return {
      password: "",
      confirmPassword : "",
      ssh: ""
    }
  },
  methods: {
    submit: function() {
      
      var details = {'password' : this.password, "confirm": this.confirmPassword, 'ssh': this.ssh}
      $.ajax({
        type: "POST",
        contentType: "application/json;charset=utf-8",
        url: "/api/root",
        traditional: "true",
        data: JSON.stringify({details}),
        dataType: "json",
        success: function(response) {
          if(response.state == 1) {
            window.location.href = "/congrats"
          } else if(response.state == 0) {
            alert("Passwords do not match")
          }
        },
        error: function(error) {
          console.error(error)
        }
      })
    }
  }
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
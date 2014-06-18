// checks if param is an array
function isArray (arr) {
  return Object.prototype.toString.call(arr) === '[object Array]'
}

// encode object as form
function formEncode (obj) {
  var out = []
  var add = function (k, v) {
    out.push(encodeURIComponent(k) + '=' + encodeURIComponent(v))
  }

  for (var name in obj)
    if (isArray(obj[name]))
      for (var i in obj[name])
        add(name + "[" + i + "]", obj[name][i])
    else add(name, obj[name])

  return out.join("&").replace(/%20/g, "+")
}


// ajax wrapper
function ajax(url, callbackFunction) {
  this.bindFunction = function (caller, object) {
    return function () {
      return caller.apply(object, [object])
    }
  }

  this.stateChange = function (object) {
    if (this.request.readyState == 4)
      this.callbackFunction(this.request.responseText)
  }

  this.getRequest = function () {
    if (window.ActiveXObject)
      return new ActiveXObject('Microsoft.XMLHTTP')
    else if (window.XMLHttpRequest)
      return new XMLHttpRequest()
    return false
  }

  var body = arguments[2]
  this.postBody  = body && typeof body == "object" ? formEncode(body) : body

  this.callbackFunction = callbackFunction
  this.url = url
  this.request = this.getRequest()

  if (this.request) {
    var req = this.request
    req.onreadystatechange = this.bindFunction(this.stateChange, this)

    if (this.postBody) {
      req.open("POST", url, true)
      req.setRequestHeader('X-Requested-With', 'XMLHttpRequest')
      req.setRequestHeader('Content-type', 'application/x-www-form-urlencoded')
    } else {
      req.open("GET", url, true)
    }

    req.send(this.postBody)
  }
}

function butterbar (message, type) {
  type = type ? " butterbar-" + type : ""

  document.querySelector('.butterbar .message').innerHTML = message
  var b = document.getElementsByClassName('butterbar')[0]
  b.className = "butterbar visible" + type
  setTimeout(function () {
    b.className = "butterbar" + type
  }, 4000)
}

function validateSignup () {
  var accountInput = document.getElementById("account")
  accountInput.onchange = function (e) {
    var id = e.target.value
    ajax('/session/' + id, function (res) {
      console.log(res)
      try {
        var body = JSON.parse(res)
        var hasWarning = !!~accountInput.className.indexOf("warning")

        console.log("status:", body.status)
        console.log("hasWarning", hasWarning)
        if (body.status == "failed") {
          !hasWarning && (accountInput.className += " warning")
          return console.log("Account # Invalid: ", body.error)
        }

        if (body.status == "found") {
          hasWarning && (accountInput.className = accountInput.className.replace(/warning/, ''))
          return console.log("Account # Valid") // fill in other fields
        }

      } catch (e) {
        console.log("invalid response", e)
      }
    })
  }
}

// // 'routes'
// var routes = {
//   "signup": validateSignup,
//   "home": function () {
//     console.log("you're on the homepage!!!")
//   }
// }
//
//
// // on document ready
// var doc = document
// var dcl = 'DOMContentLoaded'
// var loaded = /^loaded|^i|^c/.test(doc.readyState)
// var listener
// doc.addEventListener(dcl, listener = function () {
//   console.log("Dom Ready!")
//   doc.removeEventListener(dcl, listener)
//
//   for (var route in routes)
//     ~document.body.className.indexOf(route) && routes[route]()
// })

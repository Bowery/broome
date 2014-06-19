var DevController = function () {}

// constructs the url for editing a dev, adds the event listener to the submit button
DevController.prototype.init = function () {
  var self = this

  try {
    this.editUrl = '/developers/' + document.getElementsByTagName('form')[0].getAttribute('data-token')
  } catch (err) {
    butterbar('no form found', 'alert')
  }

  document.querySelector('#dev-submit').addEventListener('click', function(e) {
    e.preventDefault()

    self.editDev()
  })
}

// grabs all the information in the form and submits it
DevController.prototype.editDev = function (cb) {
  var request = new XMLHttpRequest

  var password = document.querySelector('#dev-password').value
  if (password != document.querySelector('#dev-confirm_password').value) {
    butterbar('passwords don\'t match', 'alert')
    return
  }

  var data = {
    name: document.querySelector('#dev-name').value,
    email: document.querySelector('#dev-email').value,
    password: password,
    nextPaymentTime: document.querySelector('#dev-next_payment_time').value,
    integrationEngineer: document.querySelector('#dev-integration_engineer').value
  }

  for (var field in data)
    if (!data[field])
      delete data[field]

  data.isAdmin = document.querySelector('#dev-is_admin').checked

  $.ajax({
    url: this.editUrl,
    type: 'PUT',
    data: data
  })
    .done(function(data) {
      butterbar('success', 'confirm')
    })
    .error(function(err) {
      console.log('update failed', 'alert')
    })
}

var devController = new DevController()

window.addEventListener('load', function() {
  devController.init()
}, false)

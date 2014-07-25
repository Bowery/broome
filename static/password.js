// Copyright 2014 Bowery, Inc.
/**
 * Manages password changes
 * @constructor
 */
function PasswordController () {
  this.formEl = $('.group-password .form')

  this.editUrl = '/developers/reset/' + this.formEl.data('token')
  console.log(this.editUrl)
  $('.group-password .btn-submit').click(this.editPassword.bind(this))
}

/**
 * Grabs all the information in the form and submits it.
 * @param {Event} e
 */
PasswordController.prototype.editPassword = function (e) {
  e.preventDefault()

  var ps = document.getElementsByClassName("password")
  if (ps[0].value != ps[1].value)
    return butterbar("passwords don't match", "alert")

  var data = $(this.formEl).serialize()
  console.log(data)
  var payload = {
    url: this.editUrl,
    type: 'PUT',
    data: data
  }
  $.ajax(payload)
    .done(butterbar.bind(this, 'Update Successful.', 'confirm'))
    .error(function(err) {
      console.log(err)
      butterbar(err.responseJSON.err, 'alert')
    })
}

$(document).ready(function () {
  var pc = new PasswordController()
})

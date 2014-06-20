// Copyright 2014 Bowery, Inc.
/**
 * Manages Developer Lyfesykal
 * @constructor
 */
function DevController () {
  // memoization ftw
  this.formEl = $('.group-developer .form')

  this.editUrl = '/developers/' + this.formEl.data('token')
  console.log(this.editUrl)
  $('.group-developer .btn-submit').click(this.editDev.bind(this))
}

/**
 * Grabs all the information in the form and submits it.
 * @param {Event} e
 */
DevController.prototype.editDev = function (e) {
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
    .error(butterbar.bind(this, 'Update Failed.', 'alert'))
}

$(document).ready(function () {
  var dc = new DevController()
})

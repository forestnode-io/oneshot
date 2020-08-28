(defun oneshot ()
  "Download the current buffer over HTTP."
  (interactive)
  (shell-command-on-region
    (point-min) (point-max)
    "oneshot -q"))

(defun oneshot-view ()
  "View the current buffer in a browser."
  (interactive)
  (shell-command-on-region
    (point-min) (point-max)
    "oneshot -q"))

(defun oneshot-upload ()
  "Upload text into the current buffer over HTTP."
  (interactive)
  (insert
    (shell-command-to-string "oneshot -u")))

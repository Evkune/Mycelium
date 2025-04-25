**Fonctions du VPS**


# notification-to-rss
- Est lancé à chaque fois qu'un JSON est posté sur le topic notification du broker du VPS

- Va envoyé ce JSON sur le flux RSS


# recep-moyenne
- Est lancé à chaque fois qu'un JSON est posté sur le topic moyennestovps du broker du VPS

- Va récupérer les moyennes présentes dans ce JSON et les stocker dans la base InfluxDB du VPS
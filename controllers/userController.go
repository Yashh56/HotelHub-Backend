package controllers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/Yashh56/HotelHub/models"
	"github.com/Yashh56/HotelHub/prisma/db"
	"github.com/dgrijalva/jwt-go"
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/bcrypt"
)

type Credentials struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func Register(client *db.PrismaClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var creds Credentials
		err := json.NewDecoder(r.Body).Decode(&creds)
		if err != nil {
			http.Error(w, "Invalid input", http.StatusBadRequest)
			log.Error().Err(err).Msg("Failed to decode request body")
			return
		}

		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(creds.Password), bcrypt.DefaultCost)
		if err != nil {
			http.Error(w, "Error creating user", http.StatusInternalServerError)
			log.Error().Err(err).Msg("Failed to hash password")
			return
		}

		user, err := client.User.CreateOne(
			db.User.Email.Set(creds.Email),
			db.User.Username.Set(creds.Username),
			db.User.Password.Set(string(hashedPassword)),
		).Exec(r.Context())
		if err != nil {
			http.Error(w, "Error creating user", http.StatusInternalServerError)
			log.Error().Err(err).Msg("Failed to create user in database")
			return
		}
		w.Header().Set("Content-Type", "application/json")

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(user)
		log.Info().Msg("User created successfully")
	}
}

func Login(client *db.PrismaClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var creds Credentials
		err := json.NewDecoder(r.Body).Decode(&creds)
		if err != nil {
			http.Error(w, "Invalid input", http.StatusBadRequest)
			log.Error().Err(err).Msg("Failed to decode request body")
			return
		}

		user, err := client.User.FindUnique(
			db.User.Email.Equals(creds.Email),
		).Exec(r.Context())
		if err != nil {
			http.Error(w, "User not found", http.StatusUnauthorized)
			log.Error().Err(err).Msg("User not found")
			return
		}

		err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(creds.Password))
		if err != nil {
			http.Error(w, "Invalid credentials", http.StatusUnauthorized)
			log.Error().Err(err).Msg("Invalid password")
			return
		}

		expirationTime := time.Now().Add(24 * time.Hour)
		claims := &models.Claims{
			UserId: user.ID,
			StandardClaims: jwt.StandardClaims{
				ExpiresAt: expirationTime.Unix(),
			},
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, err := token.SignedString(models.JWTKey)
		if err != nil {
			http.Error(w, "Error logging in", http.StatusInternalServerError)
			log.Error().Err(err).Msg("Failed to sign token")
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:    "token",
			Value:   tokenString,
			Expires: expirationTime,
		})

		response := map[string]interface{}{
			"userId": user.ID,
			"token":  tokenString,
		}
		w.Header().Set("Content-Type", "application/json")

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
		log.Info().Msg("User logged in successfully")
	}
}
